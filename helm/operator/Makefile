all: tar

crds:
	cd ../../go/src/operator && $(MAKE) crds
	cp ../../go/src/operator/deploy/crds/*_crd.yaml chart/templates

tar: crds
	cd chart && tar czvf ../../../out/operator.tgz . 
