Feature: ELSA Client Message Processing

  Background:
    Given the system is configured from "elsa_services.json"

  Scenario: Successfully process a client request and acceptance
    Given T2S prepares an initial client request message using template "initial_client_request.xml" with values:
      | Field           | Value        |
      | TransactionId   | TXN_SUCCESS_001 |
      | ClientName      | Test Client A|
    When T2S sends the prepared message to the "t2sClientRequestQueueName" queue with correlation ID "TXN_SUCCESS_001"
    Then ELSA instruction "TXN_SUCCESS_001" should have status "created" within configured polling limits
    Given T2S prepares an acceptance message using template "t2s_acceptance.xml" with values:
      | Field         | Value        |
      | TransactionId | TXN_SUCCESS_001 |
    When T2S sends the prepared message to the "t2sAcceptanceQueueName" queue with correlation ID "TXN_SUCCESS_001"
    # Add a step here to check final status if needed, e.g., "ACCEPTED_BY_T2S"

  Scenario: Client request where ELSA never reaches "created" status (timeout)
    Given T2S prepares an initial client request message using template "initial_client_request.xml" with values:
      | Field           | Value        |
      | TransactionId   | TXN_TIMEOUT_002 |
      | ClientName      | Test Client B|
    When T2S sends the prepared message to the "t2sClientRequestQueueName" queue with correlation ID "TXN_TIMEOUT_002"
    Then ELSA instruction "TXN_TIMEOUT_002" should NOT have status "created" within configured polling limits
    # This scenario assumes the mock API for TXN_TIMEOUT_002 will never return "created"
