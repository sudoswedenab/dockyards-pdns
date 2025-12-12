package main

//go:generate go tool controller-gen rbac:roleName=dockyards-pdns crd object paths="./..."
