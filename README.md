
canarycage
====

[![CircleCI](https://circleci.com/gh/loilo-inc/canarycage.svg?style=svg&circle-token=b3c00e40df37ed80bc83670173d32f2e5708d25f)](https://circleci.com/gh/loilo-inc/canarycage)
[![codecov](https://codecov.io/gh/loilo-inc/canarycage/branch/master/graph/badge.svg?token=WRW1qemxSR)](https://codecov.io/gh/loilo-inc/canarycage)
![https://img.shields.io/github/tag/loilo-inc/canarycage.svg](https://img.shields.io/github/tag/loilo-inc/canarycage.svg)
[![license](https://img.shields.io/github/license/loilo-inc/canarycage.svg)](https://github.com/loilo-inc/canarycage)

A deployment tool for AWS ECS that focuses on service availability

## Description

canarycage (aka. `cage`) is a deployment tool for AWS ECS that focuses on service availability.  
In various cases of deployment to ECS, your service get broken.

What kind of status is *broken* about service?  
It is, definitely, healthy targets being gone. 

## Install

via go get 

```bash
$ go get -u github.com/loilo-inc/canarycage/cli/cage 
```

or binary from Github Releases

```
$ curl -oL https://github.com/loilo-inc/canarycage/releases/download/${VERSION}/canarycage_{linux|darwin}_{amd64|386}.zip
$ unzip canarycage_linux_amd64.zip
$ chmod +x cage
$ mv cage /usr/local/bin/cage
```

## Definition Files

cage has just 2 commands, `up` and `rollout`.  
Both commands need just 2 files, `service.json` and `task-definition.json`.

`service.json` and `task-definition.json` are declarative service definition for ECS Service.  
If you have used `ecs-cli` and are familiar with `docker-compose.yml` in development, unfortunately, you cannot use those files for canarycage.  
`docker-compose.yml` is a file just for Docker, not for ECS. There are various additional configurations for deploying you docker container to ECS and eventually you realize to have to fill complete Service/Task definition files in JSON format. 
cage use those json files compatible with aws-cli and aws-sdk.

**service.json**

service.json is json version of input for `aws ecs create-service`. You can get complete template for this file by adding `--generate-cli-skeleton` to former command.

**task-definition.json**

task-definition.json is also for `aws ecs register-task-definition`.


## Usage

### up

`up` command will register new task-definition and create service that is attached to new task-definition.
The most simple usage is as belows:

```
$ cage up --region us-west-2 ./deploy
```

In this usage, some directory and files are required:

![https://gyazo.com/92caff1d830aa3899dc57f3175c6d36e.png](https://gyazo.com/92caff1d830aa3899dc57f3175c6d36e.png)

The first argument is a directory path that contains both `service.json` and `task-definition.json` files.   
We recommend you to manage those deploy files with VCS to keep service deployment idempotent.

### rollout

`rollout` command will take some steps and eventually replace all tasks of service into new tasks.  
Basic usage is same as `up` command.

#### Fargate
 
```bash
$ cage rollout --region us-west-2 ./deploy
```

#### EC2 ECS
On EC2 ECS, you must specify EC2 Instance ID or full ARN for placing canary task 

```bash
$ cage rollout --region us-west-2 --canaryInstanceArn i-abcdef123456
```

#### Without definition files

You can execute the command without definition files by passing full options. 

```bash
$ cage rollout \
    --region us-west-2 \
    --cluster my-cluster \
    --service my-service \
    --taskDefinitionArn my-service:100 \
    --canaryInstanceArn i-abcdef123456
```

`rollout` command is the core feature of canarycage.
 It makes ECS's deployment safe, avoiding entire service go down.

Rolling out in canarycage follows several steps:

- Register new task definition (task-definition-next) with `task-definition.json`  
- Start canary task (`task-canary`) with identical networking configurations to existing service with task-definition-next  
- Wait until `task-canary` become to be running
- Register `task-canary` to target group of existing service
- Wait until `task-canary` is registered to target group and it become to be healthy
- Update existing service's task definition to task-definition-next
- Wait until service become to be stable
- Stop `task-canary`
- Complete! 😇

## Motivation

By creating canary service with identical service definition, 
you can avoid an unexpected service down. 

ECS Service is verrrrrrry fragile because it is composed of not only Docker image but a lot of VPC related resources, ECS specific configurations, and environment variable or entry script that runs in only production environment.

Those resources are often managed separately from codes, by terraform os AWS Console and that mismatch sometimes will cause service down. All cases below are actual outage that we got in development phase (fortunately not in production)

- If one of essential containers is not up, service never become stable.
- If security groups attached to service are not correctly configured, ALB can't find service task and healthy targets in target group are gone.     
- If container that receives health check from ALB will not up during ALB's health check interval and threshold, ECS will terminate tasks and healthy tasks are gone.

That means there is no way to judge if next deployment will succeed or not.  
Of course, logically, there are no safe deployment. However, most of deployment failures are caused by configuration mistakes, not codes.

Checking validness of complicated AWS's resource combinations is too difficult. So we decided to check configuration's validness by creating almost identical ECS service (service-canary) before main deployment. 

It resulted in protection of healthy service during continual delivery in development phase. We could find configuration mistakes safely without healthy service down.     

## Licence

[MIT](https://github.com/tcnksm/tool/blob/master/LICENCE)

## Author

- See [Here](https://github.com/loilo-inc/canarycage/graphs/contributors)
