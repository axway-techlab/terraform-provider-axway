.PHONY: default build clean init apply scratch

default: scratch

build:
	make -C .. install

clean:
	rm -rf .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
	rm -rf .terraform

init: build
	terraform init

apply:  build
	terraform apply -auto-approve

destroy:  build
	terraform destroy -auto-approve

plan: build
	terraform plan

scratch: build clean init apply
