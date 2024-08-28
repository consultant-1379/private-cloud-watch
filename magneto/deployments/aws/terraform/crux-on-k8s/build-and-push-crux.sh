#!/bin/bash
#
# This builds and pushes a crux image to a docker registry using a
# temporary docker-machine node in AWS.
#

github_token=$1
docker_repo=$2

github_repo="github.com/erixzone/crux"
node_name=$USER-crux-build
image_name="erixzone/crux-main"

# Check for dependencies.
dependencies="docker-machine jq aws"
for dep in $dependencies ; do
    if ! which $dep >/dev/null ; then
        echo "Can't find '$dep' executable in path, can't continue."
        exit 1
    fi
done

function usage {
    echo "$0 <github token> <docker repo to push>"
    exit 1
}

# Make sure arguments are present.
if [ -z $github_token ]; then
    usage
fi

if [ -z $docker_repo ]; then
    usage
fi

# Remove docker machine when we exit, no matter what.
function cleanup {
    docker-machine rm -f -y $node_name
}
trap cleanup EXIT

# Create temporary build node using docker-machine.
echo "Creating temporary build node $node_name using docker-machine."
docker-machine create \
    --driver amazonec2 \
    $node_name || exit 1

ssh_cmd="docker-machine ssh $node_name"

# Wait until we can ssh to the box before continuing.
# This is because some versions of docker-machine have a bug that make them
# return "exit status 255" to any ssh command early in the machine's life.
# If this takes longer than 100 attempts, give up.
echo "Testing ssh connectivity."
for i in `seq 1 100`; do
    $ssh_cmd "uptime"
    if [ $? -eq 0 ]; then break; fi
    sleep 1
done

# Perform some prerequisite tasks which are required to run "make container".
echo "Installing make."
$ssh_cmd "sudo apt-get install -y make" || exit 1
echo "Adding ubuntu user to docker group."
$ssh_cmd "sudo adduser ubuntu docker" || exit 1

# Log in to ECR.
echo "Setting up authentication to Amazon ECR."
docker_login=$(aws ecr get-login --no-include-email)
if [[ ! $docker_login ]]; then
    echo "Can't get docker login command for ECR!"
    exit 1
fi

$ssh_cmd "$docker_login" || exit 1

# Check out crux.
echo "Checking out crux code."
$ssh_cmd "git init && git pull https://${github_token}@${github_repo}" || exit 1

# Run build.
echo "Running crux build."
$ssh_cmd "make container" || exit 1

# Find image id.
echo "Finding image ID of the newly-built crux image."
image_id=$( $ssh_cmd "docker images -qf reference=\"$image_name\"" )
if [[ ! $image_id ]]; then
    echo "Can't find image ID!"
    exit 1
fi

# Tag image.
echo "Tagging crux image: [id: $image_id] [repo: $docker_repo]"
$ssh_cmd "docker tag $image_id $docker_repo" || exit 1

# Push image to registry.
echo "Pushing crux image to registry."
output=$($ssh_cmd "docker push $docker_repo")
echo "$output" | tail -1 | grep latest
if [ $? -ne 0 ]; then
    echo "Couldn't push image to registry!"
    exit 1
fi

echo "Success!"
exit 0
