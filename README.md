## What is this?

An experimental test result generator from manual inputs and outputting in the [Allure2 JSON format](https://allurereport.org/docs/how-it-works-test-result-file/) as schema.

</br>

## IMPORTANT: This is a prototype to demonstrate a possible solution with test server embedding into a Github workflow or action

</br>

### General information and known issues

> - When the capture web server is started it is possible to provide a context such as @PV or @IV and it will filter all available test scenarios to only pick out the manual ones for those tags
> - This only works on test scenarios of the BDD type, so `.feature` files that is
> - At web server start it will look for `.feature` files in and in all subdirectories of the directory named `features`
> - When saving a test result and continuing, the test result is stored in a directory named `output` on the Github runner agent, and a copy of the test result file is sent to the client browser for download

> - **Known rare issue:** Sometimes (very rarely) pressing "Save and continue" will result in a **502** but it can be fixed by reloading and choosing resubmit the page (no data is lost, it is a very rare sync issue)

</br>

## Features

- ### Main screen

![image](https://github.com/user-attachments/assets/a2b97c32-9acd-4a50-8d1f-f5c7ae3a51e6)

</br>

- ### Keeps track of tests that are done and tests remaining

![image](https://github.com/user-attachments/assets/793513c5-8074-4837-81d1-92fe68ec2917)

</br>

- ### Skipping a test is possible but not selecting a test status when saving

![image](https://github.com/user-attachments/assets/7aab8360-c80d-4799-92a3-700c16d6a479)

</br>

- ### Shuts down server and completes Github workflow when all manual tests have been run

![image](https://github.com/user-attachments/assets/f1b65db6-4077-407b-bb59-f5f778041735)

</br>

- ### Test results embed images that are uploaded into the json file

![image](https://github.com/user-attachments/assets/97354eb0-8253-429e-b9cf-707d8d25eb94)

</br>

- ### All test result json files are packed and uploaded as a workflow artifact

![image](https://github.com/user-attachments/assets/1679e61a-b586-42b3-ae44-df2377587b81)

- Each file is timestamped with time of completing the test

![image](https://github.com/user-attachments/assets/f3cfa23a-5158-482a-afaf-b19c1d852864)

</br>
</br>
