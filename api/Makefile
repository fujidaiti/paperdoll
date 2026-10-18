# Generate Dart data classes from the API spec.
.PHONY: gen-dart
gen-dart:
	npx openapi-generator-cli generate -i api.yaml -g dart -o dart_binding --global-property=models,modelTests=false,modelDocs=false
	fvm dart format dart_binding

# Generate the HTML API reference from the API spec.
.PHONY: gen-docs
gen-docs:
	npx redocly build-docs api.yaml -o api.html
