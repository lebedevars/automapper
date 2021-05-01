lints:
	golangci-lint version
	golangci-lint run -c ./linters.yaml

lints_fix:
	golangci-lint run --fix -c ./linters.yaml
