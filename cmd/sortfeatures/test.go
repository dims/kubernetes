package main

import "k8s.io/component-base/featuregate"

const (
	// owner: @user1
	// This is feature A
	FeatureA featuregate.Feature = "FeatureA"

	// owner: @user3
	// This is feature C
	FeatureC featuregate.Feature = "FeatureC"

	// owner: @user2
	// This is feature Z
	FeatureZ featuregate.Feature = "FeatureZ"
)
