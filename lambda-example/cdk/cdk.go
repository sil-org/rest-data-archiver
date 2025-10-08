package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkStackProps struct {
	awscdk.StackProps
}

func NewCdkStack(scope constructs.Construct, id, stage string, props *CdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	logGroup := awslogs.NewLogGroup(stack, jsii.String("LambdaLogGroup"), &awslogs.LogGroupProps{
		LogGroupName:  jsii.Sprintf("/aws/lambda/%s-cdk", id),
		Retention:     awslogs.RetentionDays_TWO_MONTHS,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY, // Remove logs when stack is deleted
	})

	function := awslambda.NewFunction(stack, jsii.Sprintf("%s-%s", id, stage), &awslambda.FunctionProps{
		Code:          awslambda.Code_FromAsset(jsii.String("../src/bin"), nil),
		FunctionName:  jsii.Sprintf("%s-%s", id, stage),
		Handler:       jsii.String("bootstrap"),
		LoggingFormat: awslambda.LoggingFormat_JSON,
		LogGroup:      logGroup,
		Runtime:       awslambda.Runtime_PROVIDED_AL2023(),
		Timeout:       awscdk.Duration_Seconds(jsii.Number(300)),
	})

	function.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: jsii.Strings(
			"ses:SendEmail",
		),
		Resources: &[]*string{function.FunctionArn()},
	}))

	schedule := awsevents.NewRule(stack, jsii.Sprintf("%s-schedule-%s", id, stage), &awsevents.RuleProps{
		Schedule: awsevents.Schedule_Cron(&awsevents.CronOptions{
			Minute: jsii.String("10"),
			Hour:   jsii.String("9"),
		}),
	})
	schedule.AddTarget(awseventstargets.NewLambdaFunction(function, nil))

	return stack
}

func main() {
	defer jsii.Close()

	const name = "lambda-example"
	stage := getOptionalEnv("STAGE", "production")
	short := shortenStageName(stage)

	app := awscdk.NewApp(nil)
	NewCdkStack(app, name, short, &CdkStackProps{
		awscdk.StackProps{
			Env: &awscdk.Environment{
				Region: jsii.String(os.Getenv("AWS_REGION")),
			},
			Tags: &map[string]*string{
				"managed_by":        jsii.String("cdk"),
				"itse_app_name":     jsii.String(name),
				"itse_app_customer": jsii.String("gtis"),
				"itse_app_env":      jsii.String(stage),
			},
		},
	})

	app.Synth(nil)
}

func getOptionalEnv(name, fallback string) string {
	v := os.Getenv(name)
	if v == "" {
		v = fallback
	}
	return v
}

func shortenStageName(stage string) string {
	m := map[string]string{
		"production": "prod",
		"prod":       "prod",
		"develop":    "dev",
		"dev":        "dev",
		"staging":    "stg",
		"stg":        "stg",
	}

	return m[stage]
}
