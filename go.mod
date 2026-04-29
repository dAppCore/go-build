module dappco.re/go/build

go 1.26.0

require (
	github.com/Snider/Borg v0.2.0 // Note: AX-6 deferred — awaiting dappco.re/go/crypt API parity
	github.com/gin-gonic/gin v1.12.0
	github.com/gorilla/websocket v1.5.3
	github.com/kardianos/service v1.2.4
	github.com/leaanthony/debme v1.2.1
	github.com/leaanthony/gosod v1.0.4
	github.com/oasdiff/kin-openapi v0.136.1
	github.com/oasdiff/oasdiff v1.12.3
	golang.org/x/net v0.53.0
	golang.org/x/text v0.36.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go v0.123.0 // indirect
	dappco.re/go v0.9.0
	dappco.re/go/api v0.0.0
	dappco.re/go/cli v0.0.0
	dappco.re/go/i18n v0.0.0
	dappco.re/go/io v0.0.0
	dappco.re/go/log v0.0.0
	dappco.re/go/process v0.0.0
	dappco.re/go/ws v0.0.0
	github.com/TwiN/go-color v1.4.1 // indirect
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.9.2 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oasdiff/yaml v0.0.1 // indirect
	github.com/oasdiff/yaml3 v0.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
	github.com/wI2L/jsondiff v0.7.0 // indirect
	github.com/woodsbury/decimal128 v1.4.0 // indirect
	github.com/yargevad/filepathx v1.0.0 // indirect
	go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect
	golang.org/x/arch v0.25.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

require github.com/rogpeppe/go-internal v1.14.1 // indirect

replace dappco.re/go/io => ./.compat/io

replace dappco.re/go/log => ./.compat/log

replace dappco.re/go/process => ./.compat/process

replace dappco.re/go/cli => ./.compat/cli

replace dappco.re/go/i18n => ./.compat/i18n

replace dappco.re/go/api => ./.compat/api

replace dappco.re/go/ws => ./.compat/ws
