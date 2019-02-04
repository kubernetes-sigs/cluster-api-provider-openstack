package cloudinit

/*
This package contains generated go template strings.
*/

//go:generate go run ../generate.go -package-name cloudinit -input-filename assets/masterkubeadmconfig.yaml -variable-name masterKubeadmCloudConfig -output-filename zz_generated_masterkubeadmconfig.go
//go:generate go run ../generate.go -package-name cloudinit -input-filename assets/workerkubeadmconfig.yaml -variable-name workerKubeadmCloudConfig -output-filename zz_generated_workerkubeadmconfig.go

//go:generate go run ../generate.go -package-name cloudinit -input-filename assets/ubuntu.yaml -variable-name ubuntuCloudConfig -output-filename zz_generated_ubuntu_cloud_config.go
//go:generate go run ../generate.go -package-name cloudinit -input-filename assets/centos.yaml -variable-name centosCloudConfig -output-filename zz_generated_centos_cloud_config.go

//go:generate go run ../generate.go -package-name cloudinit -input-filename assets/master.yaml -variable-name masterCloudConfig -output-filename zz_generated_master.go

//go:generate go run ../generate.go -package-name cloudinit -input-filename assets/worker.yaml -variable-name workerCloudConfig -output-filename zz_generated_worker.go

//go:generate go run ../generate.go -package-name cloudinit -input-filename assets/generic.yaml -variable-name genericCloudConfig -output-filename zz_generated_generic.go
