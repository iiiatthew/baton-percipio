package config

import (
	"testing"

	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/conductorone/baton-sdk/pkg/test"
)

func TestConfigs(t *testing.T) {
	configurationSchema := field.NewConfiguration(
		ConfigurationFields,
		field.WithConstraints(FieldRelationships...),
	)

	testCases := []test.TestCase{
		{
			map[string]string{},
			false,
			"empty",
		},
		{
			map[string]string{
				"organization-id": "1",
			},
			false,
			"missing api token",
		},
		{
			map[string]string{
				"api-token": "1",
			},
			false,
			"missing organization id",
		},
		{
			map[string]string{
				"api-token":       "1",
				"organization-id": "1",
			},
			true,
			"valid",
		},
	}

	test.ExerciseTestCases(t, configurationSchema, nil, testCases)
}
