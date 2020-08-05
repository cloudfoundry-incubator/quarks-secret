# QuarksSecret

[![godoc](https://godoc.org/code.cloudfoundry.org/quarks-secret?status.svg)](https://godoc.org/code.cloudfoundry.org/quarks-secret)
[![go report card](https://goreportcard.com/badge/code.cloudfoundry.org/quarks-secret)](https://goreportcard.com/report/code.cloudfoundry.org/quarks-secret)

<img align="right" width="200" height="39" src="https://github.com/cloudfoundry-incubator/quarks-docs/raw/master/content/en/docs/cf-operator-logo.png">

----

A QuarksSecret allows the developers to deal with the management of credentials.

- QuarksSecret can be used to generate passwords, certificates and keys
- The generated credentials can be rotated by specifying its quarkssecret’s name in a configmap.
- When a certificate is generated, `QuarksSecret` ensures that a certificate signing request is generated and is approved by the Kubernetes API server

[See the official documentation for more informations](https://quarks.suse.dev/docs/components/quarkssecret/)

----


* Incubation Proposal: [Containerizing Cloud Foundry](https://docs.google.com/document/d/1_IvFf-cCR4_Hxg-L7Z_R51EKhZfBqlprrs5NgC2iO2w/edit#heading=h.lybtsdyh8res)
* Slack: #quarks-dev on <https://slack.cloudfoundry.org>
* Backlog: [Pivotal Tracker](https://www.pivotaltracker.com/n/projects/2192232)
* Docker: https://hub.docker.com/r/cfcontainerization/cf-operator/tags
* Documentation: [quarks.suse.dev](https://quarks.suse.dev)
