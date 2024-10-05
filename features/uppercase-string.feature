@URS @upper_case_feature
Feature: Upper case a string
  As a user
  I want to upper case a string via a web service
  So that I can use it in my application

  @PV @manual
  Scenario: Uppercase a string
    Given I have a string "hello"
    When I uppercase it
    Then I should get "HELLO"
