all: tar

crds:
	cd ../../go/src/edb-operator && $(MAKE) crds
	cp ../../go/src/edb-operator/deploy/crds/*_crd.yaml chart/templates

tar: crds
	cd chart && tar czvf ../../../out/edb-operator.tgz . 
