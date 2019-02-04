package userdata

/*
This package contains generated go template strings.
*/

//go:generate go run generate.go -input-filename assets/bootstrap-kubernetes.sh -variable-name bootstrapScript -output-filename zz_generated_bootstrapscript.go
//go:generate go run generate.go -input-filename assets/bootstrap-kubernetes.service -variable-name bootstrapService -output-filename zz_generated_bootstrapservice.go
