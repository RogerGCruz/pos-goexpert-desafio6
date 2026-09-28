Feature: Weather by CEP

  Scenario: Successful lookup
    Given a valid zipcode "01001000"
    When I request the weather
    Then the response status should be 200
    And the response should contain temperatures in C F and K

  Scenario: Invalid zipcode format
    Given a valid zipcode "01001"
    When I request the weather
    Then the response status should be 422

  Scenario: CEP not found
    Given a valid zipcode "99999999"
    When I request the weather
    Then the response status should be 404

  Scenario: Weather API error
    Given a valid zipcode "02002000"
    When I request the weather
    Then the response status should be 500

  Scenario: Missing WeatherAPI key
    Given the WEATHERAPI_KEY is not set
    Given a valid zipcode "01001000"
    When I request the weather
    Then the response status should be 500
    And the response body should contain "missing WEATHERAPI_KEY"
