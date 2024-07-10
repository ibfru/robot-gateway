module github.com/opensourceways/robot-gateway

go 1.20

require (
	//community-robot-lib v0.0.0-00010101000000-000000000000
	//git-platform-sdk v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	k8s.io/apimachinery v0.29.1
)

//replace (
//	community-robot-lib v0.0.0-00010101000000-000000000000 => ./community-robot-lib
//	git-platform-sdk v0.0.0-00010101000000-000000000000 => ./git-platform-sdk
//)

require golang.org/x/sys v0.15.0 // indirect
