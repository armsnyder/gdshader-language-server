// MIT License
//
// Copyright (c) 2025 Adam Snyder
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package app

import "github.com/armsnyder/gdshader-language-server/internal/lsp"

var completionItems = func() []lsp.CompletionItem {
	var items []lsp.CompletionItem

	// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#data-types
	dataTypes := map[string]string{
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

	for label, doc := range dataTypes {
		items = append(items, lsp.CompletionItem{
			Label:         label,
			Kind:          lsp.CompletionClass,
			Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
		})
	}

	// TODO(asnyder): Filter keywords based on context.

	simpleKeywords := []string{"break", "case", "continue", "default", "do", "else", "for", "if", "return", "switch", "while", "const", "struct"}

	for _, keyword := range simpleKeywords {
		items = append(items, lsp.CompletionItem{
			Label: keyword,
			Kind:  lsp.CompletionKeyword,
		})
	}

	describedKeywords := map[string]string{
		// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#precision
		"lowp":          "low precision, usually 8 bits per component mapped to 0-1",
		"mediump":       "medium precision, usually 16 bits or half float",
		"highp":         "high precision, uses full float or integer range (32 bit default)",
		"discard":       "Discards the current fragment, preventing it from being drawn. Used in fragment shaders to skip rendering under certain conditions.",
		"in":            "An agument only for reading",
		"out":           "An argument only for writing",
		"inout":         "An argument that is fully passed via reference",
		"shader_type":   "Declares the type of shader being written, such as `canvas_item`, `spatial`, or `particle`.",
		"uniform":       "Declares a variable that can be set from outside the shader",
		"varying":       "Declares a variable that is passed between vertex and fragment shaders",
		"flat":          "The value is not interpolated",
		"smooth":        "The value is interpolated in a perspective-correct fashion. This is the default.",
		"uniform_group": "Group multiple uniforms together in the inspector",
	}

	for label, doc := range describedKeywords {
		items = append(items, lsp.CompletionItem{
			Label: label,
			Kind:  lsp.CompletionKeyword,
			Documentation: &lsp.MarkupContent{
				Kind:  lsp.MarkupMarkdown,
				Value: doc,
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
		items = append(items, lsp.CompletionItem{
			Label:         label,
			Kind:          lsp.CompletionKeyword,
			Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
		})
	}

	// Function uniform hints.
	functionUniformHints := map[string]string{
		"hint_enum":  "Displays int input as a dropdown widget in the editor.",
		"hint_range": "Displays float input as a slider in the editor.",
	}

	for label, doc := range functionUniformHints {
		items = append(items, lsp.CompletionItem{
			Label:         label,
			Kind:          lsp.CompletionFunction,
			Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
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
		items = append(items, lsp.CompletionItem{
			Label:         label,
			Kind:          lsp.CompletionKeyword,
			Documentation: &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc},
		})
	}

	// Built-in variables
	// https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#built-in-variables
	// TODO(asnyder): Set variables based on shader type.

	return items
}()
