// Package docs guarda a especificação Swagger gerada por `make swagger`
// e a página do Swagger UI, ambas embutidas no binário.
package docs

import _ "embed"

// Spec é a especificação Swagger em YAML. Não edite o arquivo à mão.
//
//go:embed swagger.yaml
var Spec []byte

// UI é a página que renderiza a especificação com o Swagger UI.
//
//go:embed swagger.html
var UI []byte
