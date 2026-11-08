package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestCatalogProjectionDropsPrivateMetadata(t *testing.T) {
	var input map[string]any
	if err := json.Unmarshal([]byte(`{"data":[{"id":"fixture-image","capabilities":["image"],"input_schema":{"properties":{"prompt":{"type":"string","description":"private-fixture","examples":["private-fixture"],"channelInfo":"private-fixture"},"meta_data":{"type":"json"},"apiKey":{"type":"string"}},"constraints":{"routingMetadata":{"name":"private-fixture"},"reference_image":{"max_bytes":[123],"supplierName":"private-fixture"},"reference_transport":{"meta_data":["public_https_url"],"channelInfo":{"name":"private-fixture"}}},"operations":{"image_generation":{"request":{"method":"POST","path":"/v1/images/generations","routingMetadata":"private-fixture"},"inputs":{"image":{"field":"image","transports":["multipart_file"],"supplierId":"private-fixture"}},"internalNote":"private-fixture"}},"lifecycle":{"poll":{"method":"GET","path":"/v1/images/{image_id}","internalDeployment":"private-fixture"}},"outputSizes":["1024x1024",{"metadata":"private-fixture"}]}}]}`), &input); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(projectCatalog(input))
	for _, forbidden := range []string{"private-fixture", "apiKey", "channelInfo", "description", "examples", "routingMetadata", "supplierId", "internalDeployment"} {
		if strings.Contains(string(body), forbidden) {
			t.Errorf("catalog retained private metadata field: %s", forbidden)
		}
	}
	for _, allowed := range []string{"fixture-image", "meta_data", "max_bytes", "multipart_file", "/v1/images/generations", "/v1/images/{image_id}", "1024x1024"} {
		if !strings.Contains(string(body), allowed) {
			t.Errorf("catalog lost public parameter: %s", allowed)
		}
	}
}

func TestCatalogProjectionPreservesReviewedSchemas(t *testing.T) {
	for _, file := range []string{"../../references/media-model-catalog.json", "reviewed-video-profiles.json"} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var catalog struct {
			Models []struct {
				ID     string         `json:"id"`
				Schema map[string]any `json:"parameterSchema"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &catalog); err != nil {
			t.Fatal(err)
		}
		for _, model := range catalog.Models {
			want := reviewedMediaSchema(model.ID, model.Schema)
			result := projectCatalog(map[string]any{"data": []any{map[string]any{"id": model.ID, "capabilities": []any{"video"}, "input_schema": model.Schema}}})
			got := jsonObject(jsonArray(jsonObject(result)["data"])[0])["input_schema"]
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: public projection changed reviewed parameters", model.ID)
			}
		}
	}
}

func TestCatalogProjectionRejectsAliasesAndNestedValues(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(`{"properties":{"api_key":{"type":"string"},"API.KEY":{"type":"string"},"future_control":{"type":"integer","minimum":1,"maximum":3,"minLength":1},"prompt":{"type":"string","enum":["valid","Bearer synthetic-token","https://private.invalid",{"name":"private-fixture"}],"default":{"name":"private-fixture"}}},"operations":{"image_generation":{"request":{"method":"POST","path":"/private/implementation"}}},"constraints":{"unsupported_operations":{"future_operation":true},"parameter_modes":{"future_control":["image_generation"]}}}`), &schema); err != nil {
		t.Fatal(err)
	}
	got := projectPublicCatalogSchema(schema)
	properties := jsonObject(got["properties"])
	if properties["api_key"] != nil || properties["API.KEY"] != nil ||
		!reflect.DeepEqual(properties["future_control"], jsonObject(schema["properties"])["future_control"]) {
		t.Fatal("credential aliases leaked or public dynamic rules were lost")
	}
	if !reflect.DeepEqual(properties["prompt"], map[string]any{"type": "string", "enum": []any{"valid"}}) {
		t.Fatal("unsafe scalar or nested value escaped projection")
	}
	if !reflect.DeepEqual(got["constraints"], schema["constraints"]) {
		t.Fatal("dynamic restriction names were lost")
	}
	operation := jsonObject(jsonObject(got["operations"])["image_generation"])
	if jsonObject(operation["request"])["path"] != nil {
		t.Fatal("private route exposed")
	}
}

func TestCatalogProjectionRejectsUnknownTypesAndProtocols(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{"prompt": map[string]any{"type": "private_supplier_fixture"}},
		"operations": map[string]any{"image_generation": map[string]any{"request": map[string]any{"contentType": "private_protocol_fixture"}}},
	}
	got := projectPublicCatalogSchema(schema)
	body, _ := json.Marshal(got)
	if strings.Contains(string(body), "private_") {
		t.Fatal("unknown type or protocol crossed projection")
	}
	for _, kind := range []string{"integer", "number", "boolean", "string", "string[]", "json"} {
		for _, contentType := range []string{"application/json", "multipart/form-data"} {
			schema["properties"] = map[string]any{"custom": map[string]any{"type": kind}}
			schema["operations"] = map[string]any{"image_edit": map[string]any{"request": map[string]any{"contentType": contentType}}}
			if !reflect.DeepEqual(projectPublicCatalogSchema(schema), schema) {
				t.Fatal("supported type or protocol lost")
			}
		}
	}
}
