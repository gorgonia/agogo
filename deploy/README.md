#  cluster/image/player deployment

## what does all this do?
* create a kubernetes cluster
* create a docker image and push it to amazon ecr
* create a job on the cluster using the image from ecr
* copy random data to s3!

## requirements/tested with:

* aws access key/secret

* docker 18.03.1-ce
* kops 1.9.0
* kubectl 1.10.1
* awscli 1.15.10


## set these vars/run these commands:
export vars:
```
export AWS_DEFAULT_REGION=ap-southeast-2
export AWS_ACCESS_KEY_ID="<your key here>"
export AWS_SECRET_ACCESS_KEY="<your secret here>"
```
get container repository login:
```
aws ecr get-login --no-include-email
```
copy/paste output, should look like the following:
```
docker login -u AWS -p xxxxx https://954347443578.dkr.ecr.ap-southeast-2.amazonaws.com
```
edit vars.sh, set values
```
vi vars.sh :)
```
load env vars and deploy kubernetes cluster:
```
source vars.sh
./cluster-up.sh
```
wait for the cluster to come up:
```
kops validate cluster
```
replace user:pass in this line in selfplay/Dockerfile (while the repo is private):
```
RUN git clone https://user:pass@github.com/chewxy/agogo.git
```
build/push the docker image:
```
make cpu push
```
deploy the cpu player/s3 random data generator!
```
cd selfplay
./deploy-cpu-player.sh
```
afterwards, kill the cluster:
```
cd ..
./cluster-down.sh
```
unset the vars:
```
./unset-vars.sh
```
