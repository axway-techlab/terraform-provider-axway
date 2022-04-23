.PHONY: default build clean init apply scratch

default: scratch

build:
	make -C .. install

clean:
	rm -rf terraform.tfstate terraform.tfstate.backup

init: build
	terraform init

apply:  build
	terraform apply

destroy:  build
	terraform destroy

plan: build
	terraform plan

scratch: build clean init apply
