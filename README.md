
canarycage
====

[![CircleCI](https://circleci.com/gh/loilo-inc/canarycage.svg?style=svg&circle-token=b3c00e40df37ed80bc83670173d32f2e5708d25f)](https://circleci.com/gh/loilo-inc/canarycage)
[![codecov](https://codecov.io/gh/loilo-inc/canarycage/branch/master/graph/badge.svg?token=WRW1qemxSR)](https://codecov.io/gh/loilo-inc/canarycage)

A deployment tool for AWS ECS that focuses on service availability

## Description

canarycage (aka. `cage`) is a deployment tool for AWS ECS that focuses on service availability.  
In various cases of deployment to ECS, your service get broken.

What kind of status is *broken* about service?  
It is, definitely, healthy targets being gone. 

## Install

```bash
$ go get -u github.com/loilo-inc/canarycage/cli/cage 
```

or binary from Github Releases

```
$ curl -oL https://github.com/loilo-inc/canarycage/releases/download/${VERSION}/canarycage_${OS}_${ARCH}.zip
$ unzip canarycage_linux_amd64.zip
$ chmod +x cage
$ mv cage /usr/local/bin/cage
```

or from S3
```bash 
$ curl -oL https://s3-us-west-2.amazonaws.com/loilo-public/oss/canarycage/${VERSION}/canarycage_${OS}_${ARCH}.zip
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
$ cage up ./deploy
```

In this usage, you should create directories like:

![https://gyazo.com/92caff1d830aa3899dc57f3175c6d36e.png](https://gyazo.com/92caff1d830aa3899dc57f3175c6d36e.png)

The first argument is a directory path that contains both `service.json` and `task-definition.json` files.   
We prefer to manage those deploy files with VCS to keep service deployment idempotent.

### rollout

`rollout` command will take some steps and eventually replace all tasks of service into new tasks.  
Basic usage is same as `up` command. 
```bash
$ cage rollout ./deploy
```

`rollout` command is the core feature of canarycage.
 It makes ECS's deployment safe, avoiding entire service go down.

Rolling out in canarycage follows several steps:

- Register new task definition (task-definition-next) with `task-definition.json`  
- Create canary service (`service-canary`) with `service.json` with task-definition-next  
- Wait until `service-canary` become to be stable
- Check `service-canary`'s task is registered to TargetGroup and Waiting until it become to be healthy
- Update existing main service's task definition with task-definition-next
- Wait until rolling update finished
- Delete `service-canary`
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

[keroxp](https://github.com/keroxp)
[Yuto Urakami](https://github.com/YutoUrakami)
