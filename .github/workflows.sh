# read the workflow templates
TEMPLATE_PUBLISH=$(cat ./workflow-templates/publish.yaml)
TEMPLATE_RELEASE_GO=$(cat ./workflow-templates/release-go.yaml)
TEMPLATE_TEST_GO=$(cat ./workflow-templates/test-go.yaml)
TEMPLATE_TEST_TS=$(cat ./workflow-templates/test-ts.yaml)

# iterate each service in cmd directory
for CMD in $(ls ../cmd); do
    echo "generating workflows for cmd/${CMD}"

    PUBLISH=$(echo "${TEMPLATE_PUBLISH}" | sed "s/{{CMD}}/${CMD}/g")  # replace template cmd placeholder with cmd name
    echo "${PUBLISH}" > ./workflows/${CMD}-publish.yaml               # save workflow to workflows/{CMD}

    if [ -e ../cmd/${CMD}/main.go ] # generate this workflows only if go service
    then
#      RELEASE=$(echo "${TEMPLATE_RELEASE_GO}" | sed "s/{{CMD}}/${CMD}/g")
#      echo "${RELEASE}" > ./workflows/${CMD}-release.yaml

      TEST=$(echo "${TEMPLATE_TEST_GO}" | sed "s/{{CMD}}/${CMD}/g")
      echo "${TEST}" > ./workflows/${CMD}-test.yaml
    fi

    if [ -e ../cmd/${CMD}/package.json ]
    then
      TEST=$(echo "${TEMPLATE_TEST_TS}" | sed "s/{{CMD}}/${CMD}/g")
      echo "${TEST}" > ./workflows/${CMD}-test.yaml
    fi
done
