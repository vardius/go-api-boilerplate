# read the workflow templates
TEMPLATE_TEST=$(cat ./workflow-templates/test.yaml)
TEMPLATE_RELEASE=$(cat ./workflow-templates/release.yaml)
TEMPLATE_PUBLISH=$(cat ./workflow-templates/publish.yaml)

# iterate each service in cmd directory
for CMD in $(ls ../cmd); do
    echo "generating workflows for cmd/${CMD}"

    PUBLISH=$(echo "${TEMPLATE_PUBLISH}" | sed "s/{{CMD}}/${CMD}/g")  # replace template cmd placeholder with cmd name
    echo "${PUBLISH}" > ./workflows/${CMD}-publish.yaml               # save workflow to workflows/{CMD}

    if [ -e ../cmd/${CMD}/main.go ] # generate this workflows only if go service
    then
#      RELEASE=$(echo "${TEMPLATE_RELEASE}" | sed "s/{{CMD}}/${CMD}/g")
#      echo "${RELEASE}" > ./workflows/${CMD}-release.yaml

      TEST=$(echo "${TEMPLATE_TEST}" | sed "s/{{CMD}}/${CMD}/g")
      echo "${TEST}" > ./workflows/${CMD}-test.yaml
    fi
done
