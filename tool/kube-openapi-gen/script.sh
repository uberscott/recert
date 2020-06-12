# expected params are <group> and <version>

cd /work

GROUP=$1
VERSION=$2

# Run openapi-gen for each of your API group/version packages
kube-openapi-gen --logtostderr=true -o "" -i ./pkg/apis/$GROUP/$VERSION -O zz_generated.openapi -h /opt/boilerplate.go.txt -p ./pkg/apis/$GROUP/$VERSION -r "-"
