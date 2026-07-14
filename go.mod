module github.com/Axemere-LLC/gismo-agent-go

go 1.26.5

replace github.com/Axemere-LLC/gismo-sdk-go => ../gismo-sdk-go

replace github.com/Axemere-LLC/gismo-contracts => ../gismo-contracts

require (
	github.com/Axemere-LLC/gismo-sdk-go v0.0.0-00010101000000-000000000000
	github.com/modelcontextprotocol/go-sdk v1.6.1
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)
