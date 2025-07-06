// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package app

import (
	"slices"
	"unicode"
	"unicode/utf8"

	"github.com/armsnyder/gdshader-language-server/internal/lsp"
	"github.com/samber/lo"
)

type completionContext struct {
	shaderType   string
	functionName string
	lineTokens   []string
}

func (c completionContext) lastToken() string {
	if len(c.lineTokens) == 0 {
		return ""
	}
	return c.lineTokens[len(c.lineTokens)-1]
}

type completionPredicate func(c completionContext) bool

type completionItemPredicate struct {
	predicate completionPredicate
	item      lsp.CompletionItem
}

var alwaysTrue = func(completionContext) bool {
	return true
}

func ifLastTokenOneOf(tokens ...string) completionPredicate {
	return func(c completionContext) bool {
		return slices.Contains(tokens, c.lastToken())
	}
}

func ifFirstTokenOneOf(tokens ...string) completionPredicate {
	return func(c completionContext) bool {
		if len(c.lineTokens) == 0 {
			return false
		}
		return slices.Contains(tokens, c.lineTokens[0])
	}
}

func ifIsFirst(c completionContext) bool {
	return len(c.lineTokens) == 0
}

func ifShaderType(shaderType string) completionPredicate {
	return func(c completionContext) bool {
		return c.shaderType == shaderType
	}
}

func and(predicates ...completionPredicate) completionPredicate {
	return func(c completionContext) bool {
		for _, predicate := range predicates {
			if !predicate(c) {
				return false
			}
		}
		return true
	}
}

func or(predicates ...completionPredicate) completionPredicate {
	return func(c completionContext) bool {
		for _, predicate := range predicates {
			if predicate(c) {
				return true
			}
		}
		return false
	}
}

func not(predicate completionPredicate) completionPredicate {
	return func(c completionContext) bool {
		return !predicate(c)
	}
}

func ifTokensContain(search string) completionPredicate {
	return func(c completionContext) bool {
		return slices.Contains(c.lineTokens, search)
	}
}

func inFunction(name string) completionPredicate {
	return func(c completionContext) bool {
		return c.functionName == name
	}
}

// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#data-types
var dataTypes = map[string]string{
	"void":               "Void datatype, useful only for functions that return nothing.",
	"bool":               "Boolean datatype, can only contain `true` or `false`.",
	"bvec2":              "Two-component vector of booleans.",
	"bvec3":              "Three-component vector of booleans.",
	"bvec4":              "Four-component vector of booleans.",
	"int":                "32 bit signed scalar integer.",
	"ivec2":              "Two-component vector of signed integers.",
	"ivec3":              "Three-component vector of signed integers.",
	"ivec4":              "Four-component vector of signed integers.",
	"uint":               "Unsigned scalar integer; can't contain negative numbers.",
	"uvec2":              "Two-component vector of unsigned integers.",
	"uvec3":              "Three-component vector of unsigned integers.",
	"uvec4":              "Four-component vector of unsigned integers.",
	"float":              "32 bit floating-point scalar.",
	"vec2":               "Two-component vector of floating-point values.",
	"vec3":               "Three-component vector of floating-point values.",
	"vec4":               "Four-component vector of floating-point values.",
	"mat2":               "2x2 matrix, in column major order.",
	"mat3":               "3x3 matrix, in column major order.",
	"mat4":               "4x4 matrix, in column major order.",
	"sampler2D":          "Sampler type for binding 2D textures, which are read as float.",
	"isampler2D":         "Sampler type for binding 2D textures, which are read as signed integer.",
	"usampler2D":         "Sampler type for binding 2D textures, which are read as unsigned integer.",
	"sampler2DArray":     "Sampler type for binding 2D texture arrays, which are read as float.",
	"isampler2DArray":    "Sampler type for binding 2D texture arrays, which are read as signed integer.",
	"usampler2DArray":    "Sampler type for binding 2D texture arrays, which are read as unsigned integer.",
	"sampler3D":          "Sampler type for binding 3D textures, which are read as float.",
	"isampler3D":         "Sampler type for binding 3D textures, which are read as signed integer.",
	"usampler3D":         "Sampler type for binding 3D textures, which are read as unsigned integer.",
	"samplerCube":        "Sampler type for binding Cubemaps, which are read as float.",
	"samplerCubeArray":   "Sampler type for binding Cubemap arrays, which are read as float. Only supported in Forward+ and Mobile, not Compatibility.",
	"samplerExternalOES": "External sampler type. Only supported in Compatibility/Android platform.",
}

func isLastTokenDataType(c completionContext) bool {
	_, ok := dataTypes[c.lastToken()]
	return ok
}

var simpleKeywords = []string{"break", "case", "continue", "default", "do", "else", "for", "if", "return", "switch", "while", "const", "struct"}

func sequence(keyword string, tokens ...string) completionPredicate {
	return func(c completionContext) bool {
		return keyword == tokens[len(tokens)-1] && slices.Equal(c.lineTokens[:len(c.lineTokens)-1], tokens[:len(tokens)-1])
	}
}

func isLastTokenPunctuation(c completionContext) bool {
	if len(c.lineTokens) == 0 {
		return false
	}
	lastToken := c.lastToken()
	r, _ := utf8.DecodeRuneInString(lastToken)
	return unicode.IsPunct(r)
}

var completionItems = func() []completionItemPredicate {
	var items []completionItemPredicate

	for label, doc := range dataTypes {
		items = append(items, completionItemPredicate{
			predicate: or(
				isLastTokenPunctuation,
				ifLastTokenOneOf("uniform", "varying", "in", "out", "inout", "flat", "smooth", "lowp", "mediump", "highp"),
			),
			item: lsp.CompletionItem{
				Label:         label,
				Kind:          lsp.CompletionClass,
				Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
			},
		})
	}

	for _, keyword := range simpleKeywords {
		items = append(items, completionItemPredicate{
			predicate: or(
				not(or(isLastTokenDataType, ifLastTokenOneOf(simpleKeywords...))),
				sequence(keyword, "else", "if"),
			),
			item: lsp.CompletionItem{
				Label: keyword,
				Kind:  lsp.CompletionKeyword,
			},
		})
	}

	type predicateDescription struct {
		predicate   completionPredicate
		description string
	}

	isFirstInArgument := ifLastTokenOneOf("(", ",")

	describedKeywords := map[string]predicateDescription{
		// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#precision
		"lowp":    {alwaysTrue, "low precision, usually 8 bits per component mapped to 0-1"},
		"mediump": {alwaysTrue, "medium precision, usually 16 bits or half float"},
		"highp":   {alwaysTrue, "high precision, uses full float or integer range (32 bit default)"},
		// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#discarding
		"discard":        {ifIsFirst, "Discards the current fragment, preventing it from being drawn. Used in fragment shaders to skip rendering under certain conditions."},
		"in":             {isFirstInArgument, "An argument only for reading"},
		"out":            {isFirstInArgument, "An argument only for writing"},
		"inout":          {isFirstInArgument, "An argument that is fully passed via reference"},
		"shader_type":    {ifIsFirst, "Declares the type of shader being written, such as `canvas_item`, `spatial`, or `particle`."},
		"render_mode":    {ifIsFirst, "Declares one or more render modes of the shader"},
		"uniform":        {ifIsFirst, "Declares a variable that can be set from outside the shader"},
		"varying":        {ifIsFirst, "Declares a variable that is passed between vertex and fragment shaders"},
		"flat":           {ifLastTokenOneOf("varying"), "The value is not interpolated"},
		"smooth":         {ifLastTokenOneOf("varying"), "The value is interpolated in a perspective-correct fashion. This is the default."},
		"group_uniforms": {ifIsFirst, "Group multiple uniforms together in the inspector"},
	}

	for label, predDesc := range describedKeywords {
		items = append(items, completionItemPredicate{
			predicate: and(predDesc.predicate, not(func(c completionContext) bool {
				_, ok := describedKeywords[c.lastToken()]
				return ok
			})),
			item: lsp.CompletionItem{
				Label: label,
				Kind:  lsp.CompletionKeyword,
				Documentation: &lsp.MarkupContent{
					Kind:  lsp.MarkupMarkdown,
					Value: predDesc.description,
				},
			},
		})
	}

	// Non-function uniform himts.
	uniformHints := map[string]string{
		"source_color":                  "Used as color.",
		"hint_normal":                   "Used as normalmap.",
		"hint_default_white":            "As value or albedo color, default to opaque white.",
		"hint_default_black":            "As value or albedo color, default to opaque black.",
		"hint_default_transparent":      "As value or albedo color, default to transparent black.",
		"hint_anisotropy":               "As flowmap, default to right.",
		"repeat_enable":                 "Enabled texture repeating.",
		"repeat_disable":                "Disabled texture repeating.",
		"hint_screen_texture":           "Texture is the screen texture.",
		"hint_depth_texture":            "Texture is the depth texture.",
		"hint_normal_roughness_texture": "Texture is the normal roughness texture (only supported in Forward+).",
	}

	roughnessHints := []string{"r", "g", "b", "a", "normal", "gray"}
	for _, channel := range roughnessHints {
		uniformHints["hint_roughness_"+channel] = "Used for roughness limiter on import (attempts reducing specular aliasing). `_normal` is a normal map that guides the roughness limiter, with roughness increasing in areas that have high-frequency detail."
	}

	filterHints := []string{"nearest", "linear", "nearest_mipmap_nearest", "linear_mipmap_nearest", "nearest_mipmap_linear", "linear_mipmap_linear"}
	for _, filter := range filterHints {
		uniformHints["hint_filter_"+filter] = "Enabled specified texture filtering."
	}

	for label, doc := range uniformHints {
		items = append(items, completionItemPredicate{
			predicate: and(ifFirstTokenOneOf("uniform"), ifTokensContain(":")),
			item: lsp.CompletionItem{
				Label:         label,
				Kind:          lsp.CompletionKeyword,
				Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
			},
		})
	}

	// Function uniform hints.
	functionUniformHints := map[string]string{
		"hint_enum":  "Displays int input as a dropdown widget in the editor.",
		"hint_range": "Displays float input as a slider in the editor.",
	}

	for label, doc := range functionUniformHints {
		items = append(items, completionItemPredicate{
			predicate: and(ifFirstTokenOneOf("uniform"), ifTokensContain(":")),
			item: lsp.CompletionItem{
				Label:         label,
				Kind:          lsp.CompletionFunction,
				Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
			},
		})
	}

	// Shader types
	shaderTypes := map[string]string{
		"canvas_item": "Canvas item shader, used for 2D rendering.",
		"spatial":     "Spatial shader, used for 3D rendering.",
		"particles":   "Particle shader, used for particle systems.",
		"sky":         "Sky shader, used for rendering skyboxes or skydomes.",
		"fog":         "Fog shader, used for rendering fog effects.",
	}
	for label, doc := range shaderTypes {
		items = append(items, completionItemPredicate{
			predicate: ifLastTokenOneOf("shader_type"),
			item: lsp.CompletionItem{
				Label:         label,
				Kind:          lsp.CompletionKeyword,
				Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
			},
		})
	}

	// Built-in variables
	// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#built-in-variables
	// TODO(asnyder): Set variables based on shader type.

	type renderMode struct {
		mode        string
		description string
	}

	makeRenderModeItems := func(modes ...renderMode) []completionItemPredicate {
		return lo.Map(modes, func(mode renderMode, _ int) completionItemPredicate {
			return completionItemPredicate{
				predicate: ifFirstTokenOneOf("render_mode"),
				item: lsp.CompletionItem{
					Label:         mode.mode,
					Kind:          lsp.CompletionKeyword,
					Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: mode.description},
				},
			}
		})
	}

	type constant struct {
		name      string
		shortDesc string
		longDesc  string
	}

	makeConstantItems := func(constants ...constant) []completionItemPredicate {
		return lo.Map(constants, func(c constant, _ int) completionItemPredicate {
			return completionItemPredicate{
				predicate: alwaysTrue,
				item: lsp.CompletionItem{
					Label:         c.name,
					Kind:          lsp.CompletionConstant,
					Detail:        c.shortDesc,
					Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: c.longDesc},
				},
			}
		})
	}

	makeFunctionConstantItems := func(name string, constants ...constant) []completionItemPredicate {
		return lo.Map(constants, func(c constant, _ int) completionItemPredicate {
			return completionItemPredicate{
				predicate: inFunction(name),
				item: lsp.CompletionItem{
					Label:         c.name,
					Kind:          lsp.CompletionConstant,
					Detail:        c.shortDesc,
					Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: c.longDesc},
				},
			}
		})
	}

	byShaderType := map[string][]completionItemPredicate{
		"spatial": append(
			// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/canvas_item_shader.html#render-modes
			makeRenderModeItems(
				renderMode{"blend_mix", "Mix blend mode (alpha is transparency), default."},
				renderMode{"blend_add", "Additive blend mode."},
				renderMode{"blend_sub", "Subtractive blend mode."},
				renderMode{"blend_mul", "Multiplicative blend mode."},
				renderMode{"blend_premul_alpha", "Premultiplied alpha blend mode (fully transparent = add, fully opaque = mix)."},
				renderMode{"depth_draw_opaque", "Only draw depth for opaque geometry (not transparent)."},
				renderMode{"depth_draw_always", "Always draw depth (opaque and transparent)."},
				renderMode{"depth_draw_never", "Never draw depth."},
				renderMode{"depth_prepass_alpha", "Do opaque depth pre-pass for transparent geometry."},
				renderMode{"depth_test_disabled", "Disable depth testing."},
				renderMode{"sss_mode_skin", "Subsurface Scattering mode for skin (optimizes visuals for human skin, e.g. boosted red channel)."},
				renderMode{"cull_back", "Cull back-faces (default)."},
				renderMode{"cull_front", "Cull front-faces."},
				renderMode{"cull_disabled", "Culling disabled (double sided)."},
				renderMode{"unshaded", "Result is just albedo. No lighting/shading happens in material, making it faster to render."},
				renderMode{"wireframe", "Geometry draws using lines (useful for troubleshooting)."},
				renderMode{"debug_shadow_splits", "Directional shadows are drawn using different colors for each split (useful for troubleshooting)."},
				renderMode{"diffuse_burley", "Burley (Disney PBS) for diffuse (default)."},
				renderMode{"diffuse_lambert", "Lambert shading for diffuse."},
				renderMode{"diffuse_lambert_wrap", "Lambert-wrap shading (roughness-dependent) for diffuse."},
				renderMode{"diffuse_toon", "Toon shading for diffuse."},
				renderMode{"specular_schlick_ggx", "Schlick-GGX for direct light specular lobes (default)."},
				renderMode{"specular_toon", "Toon for direct light specular lobes."},
				renderMode{"specular_disabled", "Disable direct light specular lobes. Doesn't affect reflected light (use `SPECULAR = 0.0` instead)."},
				renderMode{"skip_vertex_transform", "`VERTEX`, `NORMAL`, `TANGENT`, and `BITANGENT` need to be transformed manually in the `vertex()` function."},
				renderMode{"world_vertex_coords", "`VERTEX`, `NORMAL`, `TANGENT`, and `BITANGENT` are modified in world space instead of model space."},
				renderMode{"ensure_correct_normals", "Use when non-uniform scale is applied to mesh *(note: currently unimplemented)*."},
				renderMode{"shadows_disabled", "Disable computing shadows in shader. The shader will not receive shadows, but can still cast them."},
				renderMode{"ambient_light_disabled", "Disable contribution from ambient light and radiance map."},
				renderMode{"shadow_to_opacity", "Lighting modifies the alpha so shadowed areas are opaque and non-shadowed areas are transparent. Useful for overlaying shadows onto a camera feed in AR."},
				renderMode{"vertex_lighting", "Use vertex-based lighting instead of per-pixel lighting."},
				renderMode{"particle_trails", "Enables the trails when used on particles geometry."},
				renderMode{"alpha_to_coverage", "Alpha antialiasing mode, see [this PR](https://github.com/godotengine/godot/pull/40364) for more."},
				renderMode{"alpha_to_coverage_and_one", "Alpha antialiasing mode, see [this PR](https://github.com/godotengine/godot/pull/40364) for more."},
				renderMode{"fog_disabled", "Disable receiving depth-based or volumetric fog. Useful for `blend_add` materials like particles."},
			),
			append(
				// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/canvas_item_shader.html#global-built-ins
				makeConstantItems(
					constant{"TIME", "in float TIME", "Global time since the engine has started, in seconds. It repeats after every `3,600` seconds (which can be changed with the `rollover` setting). It's affected by `time_scale` but not by pausing. If you need a `TIME` variable that is not affected by time scale, add your own global shader uniform and update it each frame."},
					constant{"PI", "in float PI", "A `PI` constant (`3.141592`). A ratio of a circle's circumference to its diameter and amount of radians in half turn."},
					constant{"TAU", "in float TAU", "A `TAU` constant (`6.283185`). An equivalent of `PI * 2` and amount of radians in full turn."},
					constant{"E", "in float E", "An `E` constant (`2.718281`). Euler's number and a base of the natural logarithm."},
				),
				append(
					// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/canvas_item_shader.html#vertex-built-ins
					makeFunctionConstantItems("vertex",
						constant{"MODEL_MATRIX", "in mat4 MODEL_MATRIX", "Local space to world space transform. World space is the coordinates you normally use in the editor."},
						constant{"CANVAS_MATRIX", "in mat4 CANVAS_MATRIX", "World space to canvas space transform. In canvas space the origin is the upper-left corner of the screen and coordinates ranging from `(0.0, 0.0)` to viewport size."},
						constant{"SCREEN_MATRIX", "in mat4 SCREEN_MATRIX", "Canvas space to clip space. In clip space coordinates range from `(-1.0, -1.0)` to `(1.0, 1.0)`."},
						constant{"VIEWPORT_SIZE", "in vec2 VIEWPORT_SIZE", "Size of viewport (in pixels)."},
						constant{"VIEW_MATRIX", "in mat4 VIEW_MATRIX", "World space to view space transform."},
						constant{"INV_VIEW_MATRIX", "in mat4 INV_VIEW_MATRIX", "View space to world space transform."},
						constant{"MAIN_CAM_INV_VIEW_MATRIX", "in mat4 MAIN_CAM_INV_VIEW_MATRIX", "View space to world space transform of camera used to draw the current viewport."},
						constant{"INV_PROJECTION_MATRIX", "in mat4 INV_PROJECTION_MATRIX", "Clip space to view space transform."},
						constant{"NODE_POSITION_WORLD", "in vec3 NODE_POSITION_WORLD", "Node position, in world space."},
						constant{"NODE_POSITION_VIEW", "in vec3 NODE_POSITION_VIEW", "Node position, in view space."},
						constant{"CAMERA_POSITION_WORLD", "in vec3 CAMERA_POSITION_WORLD", "Camera position, in world space."},
						constant{"CAMERA_DIRECTION_WORLD", "in vec3 CAMERA_DIRECTION_WORLD", "Camera direction, in world space."},
						constant{"CAMERA_VISIBLE_LAYERS", "in uint CAMERA_VISIBLE_LAYERS", "Cull layers of the camera rendering the current pass."},
						constant{"INSTANCE_ID", "in int INSTANCE_ID", "Instance ID for instancing."},
						constant{"INSTANCE_CUSTOM", "in vec4 INSTANCE_CUSTOM", "Instance custom data (for particles, mostly)."},
						constant{"VIEW_INDEX", "in int VIEW_INDEX", "`VIEW_MONO_LEFT` (`0`) for Mono (not multiview) or left eye, `VIEW_RIGHT` (`1`) for right eye."},
						constant{"VIEW_MONO_LEFT", "in int VIEW_MONO_LEFT", "Constant for Mono or left eye, always `0`."},
						constant{"VIEW_RIGHT", "in int VIEW_RIGHT", "Constant for right eye, always `1`."},
						constant{"EYE_OFFSET", "in vec3 EYE_OFFSET", "Position offset for the eye being rendered. Only applicable for multiview rendering."},
						constant{"VERTEX", "inout vec3 VERTEX", "Position of the vertex, in model space. In world space if `world_vertex_coords` is used."},
						constant{"VERTEX_ID", "in int VERTEX_ID", "The index of the current vertex in the vertex buffer."},
						constant{"NORMAL", "inout vec3 NORMAL", "Normal in model space. In world space if `world_vertex_coords` is used."},
						constant{"TANGENT", "inout vec3 TANGENT", "Tangent in model space. In world space if `world_vertex_coords` is used."},
						constant{"BINORMAL", "inout vec3 BINORMAL", "Binormal in model space. In world space if `world_vertex_coords` is used."},
						constant{"POSITION", "out vec4 POSITION", "If written to, overrides final vertex position in clip space."},
						constant{"UV", "inout vec2 UV", "UV main channel."},
						constant{"UV2", "inout vec2 UV2", "UV secondary channel."},
						constant{"COLOR", "inout vec4 COLOR", "Color from vertices."},
						constant{"ROUGHNESS", "out float ROUGHNESS", "Roughness for vertex lighting."},
						constant{"POINT_SIZE", "inout float POINT_SIZE", "Point size for point rendering."},
						constant{"MODELVIEW_MATRIX", "inout mat4 MODELVIEW_MATRIX", "Model/local space to view space transform (use if possible)."},
						constant{"MODELVIEW_NORMAL_MATRIX", "inout mat3 MODELVIEW_NORMAL_MATRIX", ""},
						constant{"MODEL_NORMAL_MATRIX", "in mat3 MODEL_NORMAL_MATRIX", ""},
						constant{"PROJECTION_MATRIX", "inout mat4 PROJECTION_MATRIX", "View space to clip space transform."},
						constant{"BONE_INDICES", "in uvec4 BONE_INDICES", ""},
						constant{"BONE_WEIGHTS", "in vec4 BONE_WEIGHTS", ""},
						constant{"CUSTOM0", "in vec4 CUSTOM0", "Custom value from vertex primitive. When using extra UVs, `xy` is UV3 and `zw` is UV4."},
						constant{"CUSTOM1", "in vec4 CUSTOM1", "Custom value from vertex primitive. When using extra UVs, `xy` is UV5 and `zw` is UV6."},
						constant{"CUSTOM2", "in vec4 CUSTOM2", "Custom value from vertex primitive. When using extra UVs, `xy` is UV7 and `zw` is UV8."},
						constant{"CUSTOM3", "in vec4 CUSTOM3", "Custom value from vertex primitive."},
					),
					append(
						// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/canvas_item_shader.html#fragment-built-ins
						makeFunctionConstantItems("fragment",
							constant{"VIEWPORT_SIZE", "in vec2 VIEWPORT_SIZE", "Size of viewport (in pixels)."},
							constant{"FRAGCOORD", "in vec4 FRAGCOORD", "Coordinate of pixel center in screen space. `xy` specifies position in window (origin is lower-left). `z` is fragment depth and output unless `DEPTH` is written."},
							constant{"FRONT_FACING", "in bool FRONT_FACING", "`true` if current face is front facing, `false` otherwise."},
							constant{"VIEW", "in vec3 VIEW", "Normalized vector from fragment position to camera (in view space)."},
							constant{"UV", "in vec2 UV", "UV that comes from the `vertex()` function."},
							constant{"UV2", "in vec2 UV2", "UV2 that comes from the `vertex()` function."},
							constant{"COLOR", "in vec4 COLOR", "COLOR that comes from the `vertex()` function."},
							constant{"POINT_COORD", "in vec2 POINT_COORD", "Point coordinate for drawing points with `POINT_SIZE`."},
							constant{"MODEL_MATRIX", "in mat4 MODEL_MATRIX", "Model/local space to world space transform."},
							constant{"MODEL_NORMAL_MATRIX", "in mat3 MODEL_NORMAL_MATRIX", "`transpose(inverse(mat3(MODEL_MATRIX)))` for non-uniform scale. Matches `MODEL_MATRIX` otherwise."},
							constant{"VIEW_MATRIX", "in mat4 VIEW_MATRIX", "World space to view space transform."},
							constant{"INV_VIEW_MATRIX", "in mat4 INV_VIEW_MATRIX", "View space to world space transform."},
							constant{"PROJECTION_MATRIX", "in mat4 PROJECTION_MATRIX", "View space to clip space transform."},
							constant{"INV_PROJECTION_MATRIX", "in mat4 INV_PROJECTION_MATRIX", "Clip space to view space transform."},
							constant{"NODE_POSITION_WORLD", "in vec3 NODE_POSITION_WORLD", "Node position, in world space."},
							constant{"NODE_POSITION_VIEW", "in vec3 NODE_POSITION_VIEW", "Node position, in view space."},
							constant{"CAMERA_POSITION_WORLD", "in vec3 CAMERA_POSITION_WORLD", "Camera position, in world space."},
							constant{"CAMERA_DIRECTION_WORLD", "in vec3 CAMERA_DIRECTION_WORLD", "Camera direction, in world space."},
							constant{"CAMERA_VISIBLE_LAYERS", "in uint CAMERA_VISIBLE_LAYERS", "Cull layers of the camera rendering the current pass."},
							constant{"VERTEX", "in vec3 VERTEX", "`VERTEX` from `vertex()` transformed into view space. May differ if `skip_vertex_transform` is enabled."},
							constant{"LIGHT_VERTEX", "inout vec3 LIGHT_VERTEX", "Writable version of `VERTEX` for lighting calculations. Does not change fragment position."},
							constant{"VIEW_INDEX", "in int VIEW_INDEX", "`VIEW_MONO_LEFT` (0) or `VIEW_RIGHT` (1) for stereo rendering."},
							constant{"VIEW_MONO_LEFT", "in int VIEW_MONO_LEFT", "Constant for Mono or left eye, always `0`."},
							constant{"VIEW_RIGHT", "in int VIEW_RIGHT", "Constant for right eye, always `1`."},
							constant{"EYE_OFFSET", "in vec3 EYE_OFFSET", "Position offset for the eye being rendered in multiview rendering."},
							constant{"SCREEN_UV", "in vec2 SCREEN_UV", "Screen UV coordinate for current pixel."},
							constant{"DEPTH", "out float DEPTH", "Custom depth value `[0.0, 1.0]`. Must be set in all branches if written."},
							constant{"NORMAL", "inout vec3 NORMAL", "Normal from `vertex()`, in view space (unless `skip_vertex_transform` is used)."},
							constant{"TANGENT", "inout vec3 TANGENT", "Tangent from `vertex()`, in view space (unless `skip_vertex_transform` is used)."},
							constant{"BINORMAL", "inout vec3 BINORMAL", "Binormal from `vertex()`, in view space (unless `skip_vertex_transform` is used)."},
							constant{"NORMAL_MAP", "out vec3 NORMAL_MAP", "Set normal here when reading from a texture instead of using `NORMAL`."},
							constant{"NORMAL_MAP_DEPTH", "out float NORMAL_MAP_DEPTH", "Depth from `NORMAL_MAP`. Defaults to `1.0`."},
							constant{"ALBEDO", "out vec3 ALBEDO", "Base color (default white)."},
							constant{"ALPHA", "out float ALPHA", "Alpha value `[0.0, 1.0]`. Triggers transparency pipeline if used."},
							constant{"ALPHA_SCISSOR_THRESHOLD", "out float ALPHA_SCISSOR_THRESHOLD", "Alpha discard threshold."},
							constant{"ALPHA_HASH_SCALE", "out float ALPHA_HASH_SCALE", "Alpha hash dither scale (higher = more visible pixels)."},
							constant{"ALPHA_ANTIALIASING_EDGE", "out float ALPHA_ANTIALIASING_EDGE", "Alpha to coverage antialiasing edge threshold. Requires `alpha_to_coverage` render mode."},
							constant{"ALPHA_TEXTURE_COORDINATE", "out vec2 ALPHA_TEXTURE_COORDINATE", "UV for alpha-to-coverage AA. Typically `UV * texture_size`."},
							constant{"PREMUL_ALPHA_FACTOR", "out float PREMUL_ALPHA_FACTOR", "Premultiplied alpha lighting interaction. Used with `blend_premul_alpha`."},
							constant{"METALLIC", "out float METALLIC", "Metallic value `[0.0, 1.0]`."},
							constant{"SPECULAR", "out float SPECULAR", "Specular value (default `0.5`). `0.0` disables reflections."},
							constant{"ROUGHNESS", "out float ROUGHNESS", "Roughness value `[0.0, 1.0]`."},
							constant{"RIM", "out float RIM", "Rim lighting intensity `[0.0, 1.0]`."},
							constant{"RIM_TINT", "out float RIM_TINT", "Rim tint: `0.0` = white, `1.0` = albedo."},
							constant{"CLEARCOAT", "out float CLEARCOAT", "Adds a secondary specular layer."},
							constant{"CLEARCOAT_GLOSS", "out float CLEARCOAT_GLOSS", "Glossiness of clearcoat layer."},
							constant{"ANISOTROPY", "out float ANISOTROPY", "Distortion factor for specular highlight."},
							constant{"ANISOTROPY_FLOW", "out vec2 ANISOTROPY_FLOW", "Direction of anisotropy flow (e.g. from flowmaps)."},
							constant{"SSS_STRENGTH", "out float SSS_STRENGTH", "Subsurface scattering strength."},
							constant{"SSS_TRANSMITTANCE_COLOR", "out vec4 SSS_TRANSMITTANCE_COLOR", "Color for subsurface transmittance effect."},
							constant{"SSS_TRANSMITTANCE_DEPTH", "out float SSS_TRANSMITTANCE_DEPTH", "Depth for transmittance penetration."},
							constant{"SSS_TRANSMITTANCE_BOOST", "out float SSS_TRANSMITTANCE_BOOST", "Boost to force SSS to appear even when lit."},
							constant{"BACKLIGHT", "inout vec3 BACKLIGHT", "Backlighting color for light received on opposite side of surface."},
							constant{"AO", "out float AO", "Ambient occlusion intensity (for pre-baked AO)."},
							constant{"AO_LIGHT_AFFECT", "out float AO_LIGHT_AFFECT", "How much AO dims direct lighting. `[0.0, 1.0]`."},
							constant{"EMISSION", "out vec3 EMISSION", "Emissive color. Can exceed `1.0` for HDR."},
							constant{"FOG", "out vec4 FOG", "If written to, blends final color with `FOG.rgb` using `FOG.a`."},
							constant{"RADIANCE", "out vec4 RADIANCE", "Environment map radiance override."},
							constant{"IRRADIANCE", "out vec4 IRRADIANCE", "Environment map irradiance override."},
						),
						// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/canvas_item_shader.html#light-built-ins
						makeFunctionConstantItems("light",
							constant{"VIEWPORT_SIZE", "in vec2 VIEWPORT_SIZE", "Size of viewport (in pixels)."},
							constant{"FRAGCOORD", "in vec4 FRAGCOORD", "Pixel center coordinate in screen space. `xy` is position in window, `z` is depth unless `DEPTH` is used. Origin is lower-left."},
							constant{"MODEL_MATRIX", "in mat4 MODEL_MATRIX", "Model/local space to world space transform."},
							constant{"INV_VIEW_MATRIX", "in mat4 INV_VIEW_MATRIX", "View space to world space transform."},
							constant{"VIEW_MATRIX", "in mat4 VIEW_MATRIX", "World space to view space transform."},
							constant{"PROJECTION_MATRIX", "in mat4 PROJECTION_MATRIX", "View space to clip space transform."},
							constant{"INV_PROJECTION_MATRIX", "in mat4 INV_PROJECTION_MATRIX", "Clip space to view space transform."},
							constant{"NORMAL", "in vec3 NORMAL", "Normal vector, in view space."},
							constant{"SCREEN_UV", "in vec2 SCREEN_UV", "Screen UV coordinate for current pixel."},
							constant{"UV", "in vec2 UV", "UV that comes from the `vertex()` function."},
							constant{"UV2", "in vec2 UV2", "UV2 that comes from the `vertex()` function."},
							constant{"VIEW", "in vec3 VIEW", "View vector, in view space."},
							constant{"LIGHT", "in vec3 LIGHT", "Light vector, in view space."},
							constant{"LIGHT_COLOR", "in vec3 LIGHT_COLOR", "`light_color * light_energy * PI`. Includes `PI` because physically-based models divide by `PI`."},
							constant{"SPECULAR_AMOUNT", "in float SPECULAR_AMOUNT", "`2.0 * light_specular` for Omni and Spot lights. `1.0` for Directional lights."},
							constant{"LIGHT_IS_DIRECTIONAL", "in bool LIGHT_IS_DIRECTIONAL", "`true` if this pass is a DirectionalLight3D."},
							constant{"ATTENUATION", "in float ATTENUATION", "Attenuation from distance or shadow."},
							constant{"ALBEDO", "in vec3 ALBEDO", "Base albedo color."},
							constant{"BACKLIGHT", "in vec3 BACKLIGHT", "Backlighting color."},
							constant{"METALLIC", "in float METALLIC", "Metallic factor."},
							constant{"ROUGHNESS", "in float ROUGHNESS", "Roughness factor."},
							constant{"DIFFUSE_LIGHT", "out vec3 DIFFUSE_LIGHT", "Diffuse light result."},
							constant{"SPECULAR_LIGHT", "out vec3 SPECULAR_LIGHT", "Specular light result."},
							constant{"ALPHA", "out float ALPHA", "Alpha value `[0.0, 1.0]`. Enables transparent pipeline if written."},
						)...,
					)...,
				)...,
			)...,
		),
	}

	for label, typeItems := range byShaderType {
		for _, item := range typeItems {
			items = append(items, completionItemPredicate{
				predicate: and(ifShaderType(label), item.predicate),
				item:      item.item,
			})
		}
	}

	return items
}()
