@URS @day_of_week_feature
Feature: day_of_week_feature
  As a user
  I want to know what day of the week it is
  So that I can plan my day

  @PV @IV @manual @pIV
  Scenario Outline: day_of_week_feature
    Given I have the date "<date>"
    When I send a request to the day endpoint with "<date>"
    Then the response status should be "200"
    And the response body should be "<day_of_week>"

    Examples:
      | date       | day_of_week |
      | 2019-01-01 | Tuesday     |
      | 2009-01-02 | Friday   |
      | 2011-01-03 | Monday    |
      | 2019-01-04 | Friday      |
      | 2023-01-05 | Thursday    |
      | 2029-01-06 | Saturday      |
      | 2034-01-07 | Saturday      |
