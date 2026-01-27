module deploy

go 1.25.5

replace p2p_bridge => ../

require (
	github.com/aws/aws-cdk-go/awscdk/v2 v2.114.1
	github.com/aws/aws-cdk-go/awscdkapigatewayv2alpha/v2 v2.114.1-alpha.0
	github.com/aws/aws-cdk-go/awscdkapigatewayv2integrationsalpha/v2 v2.114.1-alpha.0
	github.com/aws/constructs-go/constructs/v10 v10.4.4
	github.com/aws/jsii-runtime-go v1.125.0
	p2p_bridge v0.0.0-00010101000000-000000000000
)

require (
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/cdklabs/awscdk-asset-awscli-go/awscliv1/v2 v2.2.261 // indirect
	github.com/cdklabs/awscdk-asset-kubectl-go/kubectlv20/v2 v2.1.2 // indirect
	github.com/cdklabs/awscdk-asset-node-proxy-agent-go/nodeproxyagentv6/v2 v2.1.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/yuin/goldmark v1.4.13 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/tools v0.36.0 // indirect
)
