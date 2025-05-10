Imagine you are expert software architect and you got task to handle testing automation for newly developed system. 

System info:
- name ELSA
- its backend system with microservices
- services comunicate using events stored in ibm mq queues
- they share one common DB
- system status can be obrained using rest API calls
- system orchestrate communicatio between two primary systems
  - system 1 name is T2S
  - system 2 is CREATION
- each primary system send data as messages to ELSA and ELSA sends mesasges back to it in dedicated queues

Business flow of messages are:
- T2S send client copy request message to ELSA
- ELSA waits till T2S accept request, this is done by T2S send accept/reject message to ELSA
- ELSA can send request on behalf of client to Creation system
- Creation system validate request and send back accept/reject
- on reject ELSA send cancellation to T2S
- on accept ELSA send status message to t2S informing client and matching message to T2S
- T2S validate client and ElsA matching message and if its ok send MATCH message to ELSA
- ELSA then send MACH to Creation
- Creation send back settled
- ELSA send release to T2S
- T2S send settled copy to ELSA

New testing tool requirement:
- You as expert sw architect is asked to come up with design of ELSA integration test
- treat ELSA as black box with dedicated input/output for each primary system
- This new integration testing tool have to simulate T2S and Creation message flows
- by sending mesasges simulation 100% of possible scenarios ELSA will process them based on requirements
- use ELSA rest API where neded to execute scenarios that depend on ELSA state

Elsa functionality is currently tested by postman collections checking expected ELSA behaviour using information from ELSA REST API calls.

