package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
)

// NoAssumeRoleSynthesizerProps mirrors the TS NoAssumeRoleSynthesizer in
// apps.debugjois.dev: an optional CFN execution role ARN; otherwise CFN
// executes under the calling principal.
type NoAssumeRoleSynthesizerProps struct {
	Qualifier                      *string
	CloudFormationExecutionRoleArn *string
}

type NoAssumeRoleSynthesizer interface {
	awscdk.BootstraplessSynthesizer
}

type noAssumeRoleSynthesizer struct {
	awscdk.BootstraplessSynthesizer
	cfnExecutionRoleArn *string
}

// NewNoAssumeRoleSynthesizer subclasses BootstraplessSynthesizer so the cloud
// assembly manifest emits no assumeRoleArn — `cdk deploy` uses the caller's
// current credentials directly. Optionally emits cloudFormationExecutionRoleArn
// when CloudFormationExecutionRoleArn is set.
func NewNoAssumeRoleSynthesizer(props *NoAssumeRoleSynthesizerProps) NoAssumeRoleSynthesizer {
	if props == nil {
		props = &NoAssumeRoleSynthesizerProps{}
	}
	s := &noAssumeRoleSynthesizer{
		cfnExecutionRoleArn: props.CloudFormationExecutionRoleArn,
	}
	awscdk.NewBootstraplessSynthesizer_Override(s, &awscdk.BootstraplessSynthesizerProps{
		Qualifier:                      props.Qualifier,
		CloudFormationExecutionRoleArn: props.CloudFormationExecutionRoleArn,
	})
	return s
}

// ReusableBind overrides the parent so the bound synthesizer is `s` itself
// rather than a JS-side prototype copy (Object.create(this)). The copy would
// otherwise be a fresh JSII instance with no Go override, so our Synthesize
// override would never run during App.Synth.
func (s *noAssumeRoleSynthesizer) ReusableBind(stack awscdk.Stack) awscdk.IBoundStackSynthesizer {
	s.Bind(stack)
	return s
}

func (s *noAssumeRoleSynthesizer) Synthesize(session awscdk.ISynthesisSession) {
	s.SynthesizeStackTemplate(s.BoundStack(), session)

	options := &awscdk.SynthesizeStackArtifactOptions{}
	if s.cfnExecutionRoleArn != nil {
		options.CloudFormationExecutionRoleArn = s.cfnExecutionRoleArn
	}
	s.EmitArtifact(session, options)
}
