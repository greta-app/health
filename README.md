# Health checker

## Intro and motivation
This project was intented to provide health checking logic for http endpoints. 
On one side it exposes an http address to be periodically called by the health triggering logic e.g. from k8s. On other side it calls a list of http addresses, checks response codes and evaluates response bodies. 

The checking logic is described as a json script which can be extended and adjusted evolutionarily with the corresponding application target. If a test discovers an unexpected response code or body, the health check will fail and 500 response code will be returned to the caller. 

A typical use case where this application can be used is blue green/deployments with php applications behind an nginx or apache webserver. When a green version of app is deployed and running, the health checker runs tests against its IP address plus passes Host header to allow correct routing through nginx/apache vhost configuration. In this case calling green app version won't be possible through a public domain as it's pointing to a still active green version.

Conceptually having health logic inside the PHP application itself is wrong because of following reasons:

- if you have multiple http applications which you want to test, you need to write health check code either inside of each application or use one application to test others. In the first case you need to repeat code. In the latter case if the health checking instance is unhealthy it won't be able to test other applications.

- if you use docker environment e.g. in AWS (ECS), you want be able to link your PHP container to a nginx container because this will cause a cycle error since containers cannot link each other

- conceptually it is clean to have an external service at infrastructure level to check your app, rather than let your app to be concerned about infrastructure settings like connection url to it through a webserver, docker host name, port etc. At the end your app code should be about business logic (single responsibility principle). In a microservice environment it's even more actual since health checking service a devoted instance for only this responsibility.

## Getting started

- create a script file with tests. Use `example.json` for inspiration or just the `hello world` variant below:


    # test.json
    [
      {
        "url": "https://status.aws.amazon.com/",
        "code_range": "199-299",
        "method": "GET"
      }
    ]

- pull and run the docker image with the health server at the defined port:


    docker run -it --rm -v $(pwd)/test.json:/home/ec2-user/greta/healthtests/test.json -p 127.0.0.1:8023:8023/tcp gretaapp/health /home/ec2-user/greta/health -port 8023 -scriptPath /home/ec2-user/greta/healthtests/test.json

Here is some explanation:

    docker run -it --rm : starts a docker container interactively in the current terminal and removes it after exit
     -v $(pwd)/test.json:/home/ec2-user/greta/healthtests/test.json : mounts test file from the local folder $(pwd)/test.json to the container folder at path /home/ec2-user/greta/healthtests/test.json
    -p 127.0.0.1:8023:8023/tcp : exposes port 8023 on the host machine and binds it to the port 8023 inside the docker container
    gretaapp/health : docker image path - see https://hub.docker.com/r/gretaapp/health
    /home/ec2-user/greta/health : the binary location
    -port 8023 : port where to run the health checker inside the docker container
    -scriptPath /home/ec2-user/greta/healthtests/test.json : path to the testing script
    

- you should see an output like:

`2021/01/18 12:09:06 start process on port :8023, for the script path /home/ec2-user/greta/healthtests/test.json`

- open another terminal session and run your health test:


    curl localhost:8023


- you should see something like:


    Tests success

- to see the execution details by running following command:


    docker logs $(docker ps --format {{.ID}} --filter "ancestor=gretaapp/health")

- you should see an output like:

    
    2021/01/18 12:09:06 start process on port :8023, for the script path /home/ec2-user/greta/healthtests/test.json
    2021/01/18 12:11:31 will execute 1 tests
    2021/01/18 12:11:31 Calling server with input parameters: {Url:https://status.aws.amazon.com/ ExpectedResponseCodeRange:199-299 Method:GET Contains: Headers:map[]}
    2021/01/18 12:11:33 Got response status code: '200'
    2021/01/18 12:11:33 test {Url:https://status.aws.amazon.com/ ExpectedResponseCodeRange:199-299 Method:GET Contains: Headers:map[]} execution success
    2021/01/18 12:11:33 took: 1.8651697s
    2021/01/18 12:11:33 Tests success
  
    
## Test file format

Test file format is JSON. It defines a list of tests as following:

    
    [
      {
        "url": "http://127.0.0.1:8080",
        "code_range": "199-299",
        "method": "GET",
        "contains": "Service is operating normally",
        "headers": {
          "Host": "mydomain.com"
        }
      }
    ]

### url
Defines the url to call in the test.

### code_range
Format is `{{min_response_code}}-{{max_response_code}}`. The health app will make sure that the actual response code is in the defined range `{{min_response_code}} < {{actual_response_code}} < {{max_response_code}}`. If not, the test will fail.

### method
Http method to use for testing.

### contains
The text that should be found in the response body of a corresponding url. If no match is found, the test will fail.

### headers
Key value arguments which will be used as headers to call `url`. 


The test from above will do the following:

If will call `http://127.0.0.1:8080` with a GET method and headers "Host: mydomain.com". 
If the response code is in range between 199-299 and body contains text `Service is operating normally`, the service will be successful.
