// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package app_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/armsnyder/gdshader-language-server/internal/app"
	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

func TestHandler_CompletionMatrix(t *testing.T) {
	tests := []struct {
		name        string
		document    string
		seek        func(document string) lsp.Position
		wantLabels  []string
		avoidLabels []string
	}{
		{
			name:       "SpatialRenderModes",
			document:   "shader_type spatial;\nrender_mode b",
			seek:       toAfter("b"),
			wantLabels: []string{"blend_mix", "blend_add"},
			// Present in canvas_item render modes (same prefix) but not spatial.
			avoidLabels: []string{"blend_disabled"},
		},
		{
			name:       "SpatialStencilModeKeyword",
			document:   "shader_type spatial;\nstenc",
			seek:       toAfter("stenc"),
			wantLabels: []string{"stencil_mode"},
		},
		{
			name:       "SpatialStencilModes",
			document:   "shader_type spatial;\nstencil_mode comp",
			seek:       toAfter("comp"),
			wantLabels: []string{"compare_always", "compare_equal"},
		},
		{
			name:       "CanvasRenderModes",
			document:   "shader_type canvas_item;\nrender_mode blend_d",
			seek:       toAfter("blend_d"),
			wantLabels: []string{"blend_disabled"},
		},
		{
			name:       "CanvasVertexBuiltins",
			document:   "shader_type canvas_item;\nvoid vertex() {\nINST\n}\n",
			seek:       toAfter("INST"),
			wantLabels: []string{"INSTANCE_CUSTOM"},
		},
		{
			name:       "CanvasLightBuiltins",
			document:   "shader_type canvas_item;\nvoid light() {\nLIG\n}\n",
			seek:       toAfter("LIG"),
			wantLabels: []string{"LIGHT_COLOR"},
		},
		{
			name:       "CanvasNormalBuiltins",
			document:   "shader_type canvas_item;\nvoid normal() {\nREGI\n}\n",
			seek:       toAfter("REGI"),
			wantLabels: []string{"REGION_RECT"},
		},
		{
			name:       "CanvasGlobalBuiltins",
			document:   "shader_type canvas_item;\nvoid helper() {\nTI\n}\n",
			seek:       toAfter("TI"),
			wantLabels: []string{"TIME"},
		},
		{
			name:       "CanvasSdfFunctions",
			document:   "shader_type canvas_item;\ntextu",
			seek:       toAfter("textu"),
			wantLabels: []string{"texture_sdf"},
		},
		{
			name:       "SkyRenderModes",
			document:   "shader_type sky;\nrender_mode use_h",
			seek:       toAfter("use_h"),
			wantLabels: []string{"use_half_res_pass"},
		},
		{
			name:       "SkyBuiltins",
			document:   "shader_type sky;\nvoid sky() {\nEYE\n}\n",
			seek:       toAfter("EYE"),
			wantLabels: []string{"EYEDIR"},
		},
		{
			name:       "SkyGlobalBuiltins",
			document:   "shader_type sky;\nvoid helper() {\nRAD\n}\n",
			seek:       toAfter("RAD"),
			wantLabels: []string{"RADIANCE"},
		},
		{
			name:       "FogRenderBuiltins",
			document:   "shader_type fog;\nvoid fog() {\nWORLD\n}\n",
			seek:       toAfter("WORLD"),
			wantLabels: []string{"WORLD_POSITION"},
		},
		{
			name:       "FogGlobalBuiltins",
			document:   "shader_type fog;\nvoid helper() {\nTI\n}\n",
			seek:       toAfter("TI"),
			wantLabels: []string{"TIME"},
		},
		{
			name:       "ParticlesRenderModes",
			document:   "shader_type particles;\nrender_mode keep",
			seek:       toAfter("keep"),
			wantLabels: []string{"keep_data"},
		},
		{
			name:       "ParticlesStartBuiltins",
			document:   "shader_type particles;\nvoid start() {\nRESTART_C\n}\n",
			seek:       toAfter("RESTART_C"),
			wantLabels: []string{"RESTART_CUSTOM"},
		},
		{
			name:       "ParticlesProcessBuiltins",
			document:   "shader_type particles;\nvoid process() {\nCOLL\n}\n",
			seek:       toAfter("COLL"),
			wantLabels: []string{"COLLIDED"},
		},
		{
			name:       "ParticlesStartAndProcessBuiltins",
			document:   "shader_type particles;\nvoid process() {\nRANDO\n}\n",
			seek:       toAfter("RANDO"),
			wantLabels: []string{"RANDOM_SEED"},
		},
		{
			name:       "ParticlesProcessFunctions",
			document:   "shader_type particles;\nemit_",
			seek:       toAfter("emit_"),
			wantLabels: []string{"emit_subparticle"},
		},
		{
			name:       "ParticlesGlobalBuiltins",
			document:   "shader_type particles;\nvoid helper() {\nTI\n}\n",
			seek:       toAfter("TI"),
			wantLabels: []string{"TIME"},
		},
		{
			name:       "VertexIncludesVertexOnlyBuiltins",
			document:   "shader_type spatial;\nvoid vertex() {\nBO\n}\n",
			seek:       toAfter("BO"),
			wantLabels: []string{"BONE_INDICES", "BONE_WEIGHTS"},
		},
		{
			name:        "FragmentExcludesVertexOnlyBuiltins",
			document:    "shader_type spatial;\nvoid fragment() {\nBO\n}\n",
			seek:        toAfter("BO"),
			avoidLabels: []string{"BONE_INDICES", "BONE_WEIGHTS"},
		},
		{
			name:       "FragmentIncludesFragmentOnlyBuiltins",
			document:   "shader_type spatial;\nvoid fragment() {\nAO\n}\n",
			seek:       toAfter("AO"),
			wantLabels: []string{"AO", "AO_LIGHT_AFFECT"},
		},
		{
			name:        "VertexExcludesFragmentOnlyBuiltins",
			document:    "shader_type spatial;\nvoid vertex() {\nAO\n}\n",
			seek:        toAfter("AO"),
			avoidLabels: []string{"AO", "AO_LIGHT_AFFECT"},
		},
		{
			name:       "UniformHintCompletion",
			document:   "shader_type spatial;\nuniform sampler2D tex : hint_",
			seek:       toAfter("hint_"),
			wantLabels: []string{"hint_normal"},
		},
		{
			name:       "UniformHintRangeForInt",
			document:   "shader_type spatial;\nuniform int count : hint_r",
			seek:       toAfter("hint_r"),
			wantLabels: []string{"hint_range"},
		},
		{
			name:       "UniformHintRangeForFloat",
			document:   "shader_type spatial;\nuniform float strength : hint_r",
			seek:       toAfter("hint_r"),
			wantLabels: []string{"hint_range"},
		},
		{
			name:        "UniformHintRangeNotForSampler",
			document:    "shader_type spatial;\nuniform sampler2D tex : hint_r",
			seek:        toAfter("hint_r"),
			avoidLabels: []string{"hint_range"},
		},
		{
			name:       "UniformSourceColorForVec3",
			document:   "shader_type spatial;\nuniform vec3 color : source_",
			seek:       toAfter("source_"),
			wantLabels: []string{"source_color"},
		},
		{
			name:       "UniformSourceColorForSampler",
			document:   "shader_type spatial;\nuniform sampler2D albedo : source_",
			seek:       toAfter("source_"),
			wantLabels: []string{"source_color"},
		},
		{
			name:     "UniformHintTypeRestricted",
			document: "shader_type spatial;\nuniform vec3 color : hint_",
			seek:     toAfter("hint_"),
			avoidLabels: []string{
				"hint_normal",
				"filter_nearest",
			},
		},
		{
			name:        "UniformHintRepeatExpansion",
			document:    "shader_type spatial;\nuniform sampler2D tex : repeat_",
			seek:        toAfter("repeat_"),
			wantLabels:  []string{"repeat_enable", "repeat_disable"},
			avoidLabels: []string{"repeat[_enable, _disable]"},
		},
		{
			name:       "UniformHintRoughnessExpansion",
			document:   "shader_type spatial;\nuniform sampler2D tex : hint_roughness_",
			seek:       toAfter("hint_roughness_"),
			wantLabels: []string{"hint_roughness_r", "hint_roughness_g", "hint_roughness_b", "hint_roughness_a", "hint_roughness_normal", "hint_roughness_gray"},
		},
		{
			name:        "UniformHintFilterAliases",
			document:    "shader_type spatial;\nuniform sampler2D tex : filter_",
			seek:        toAfter("filter_"),
			wantLabels:  []string{"filter_nearest", "filter_linear_mipmap"},
			avoidLabels: []string{"filter_mipmap_linear"},
		},
		{
			name:       "DataTypes",
			document:   "shader_type spatial;\nuniform ive",
			seek:       toAfter("ive"),
			wantLabels: []string{"ivec2", "ivec3"},
		},
		{
			name:       "InterpolationQualifiers",
			document:   "shader_type spatial;\nvarying sm",
			seek:       toAfter("sm"),
			wantLabels: []string{"smooth"},
		},
		{
			name:       "PrecisionQualifierCompletion",
			document:   "shader_type spatial;\nlow",
			seek:       toAfter("low"),
			wantLabels: []string{"lowp"},
		},
		{
			name:       "NestedBlockKeepsFunctionScope",
			document:   "shader_type spatial;\nvoid vertex() {\nif (true) {\nVE\n}\n}\n",
			seek:       toAfter("VE"),
			wantLabels: []string{"VERTEX"},
		},
		{
			name:        "OutsideFunctionExcludesStageBuiltins",
			document:    "shader_type spatial;\nvoid vertex() {\nVERTEX = vec3(0.0);\n}\nAO\n",
			seek:        toAfter("AO"),
			avoidLabels: []string{"AO", "AO_LIGHT_AFFECT"},
		},
		{
			name:       "HelperFunctionIncludesVertexBuiltins",
			document:   "shader_type spatial;\nvoid helper() {\nBO\n}\n",
			seek:       toAfter("BO"),
			wantLabels: []string{"BONE_INDICES", "BONE_WEIGHTS"},
		},
		{
			name:       "HelperFunctionIncludesFragmentBuiltins",
			document:   "shader_type spatial;\nvoid helper() {\nAO\n}\n",
			seek:       toAfter("AO"),
			wantLabels: []string{"AO", "AO_LIGHT_AFFECT"},
		},
		{
			name:       "SpatialGlobalBuiltins",
			document:   "shader_type spatial;\nvoid helper() {\nOUTP\n}\n",
			seek:       toAfter("OUTP"),
			wantLabels: []string{"OUTPUT_IS_SRGB"},
		},
		{
			name:       "SpatialLightBuiltins",
			document:   "shader_type spatial;\nvoid light() {\nSPECU\n}\n",
			seek:       toAfter("SPECU"),
			wantLabels: []string{"SPECULAR_AMOUNT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			g := NewWithT(t)
			var h app.Handler
			const uri = "file:///completion-matrix.gdshader"

			err := h.DidOpenTextDocument(t.Context(), lsp.DidOpenTextDocumentParams{
				TextDocument: lsp.TextDocumentItem{URI: uri, Text: tt.document},
			})
			g.Expect(err).ToNot(HaveOccurred())

			list, err := h.Completion(t.Context(), lsp.CompletionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{URI: uri},
					Position:     mustSeekPosition(t, tt.seek, tt.document),
				},
			})
			g.Expect(err).ToNot(HaveOccurred())

			labels := make([]string, 0, len(list.Items))
			for _, item := range list.Items {
				labels = append(labels, item.Label)
			}

			g.Expect(labels).To(ContainElements(tt.wantLabels))
			for _, avoid := range tt.avoidLabels {
				g.Expect(labels).ToNot(ContainElement(avoid))
			}
		})
	}
}
