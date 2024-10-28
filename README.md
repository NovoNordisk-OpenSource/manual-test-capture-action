## What is this?

An experimental test result generator from manual inputs and outputting in the [Allure2 JSON format](https://allurereport.org/docs/how-it-works-test-result-file/) as schema.

</br>

## IMPORTANT: This is an early version that demonstrates a possible solution with test server embedding into a Github workflow or action

</br>

## How to use?

**Prequiresites:**
> 1. Feature files are available in the execution context as this is a composite GitHub Action, e.g. they have been checked out at a previous step in the workflow.
> 2. The workflow has reasonable GitHub token permissions set for manupulating files and installing tools in the runner.

Embed the action call in your GitHub workflow, like this:

```yaml

```

## Notes and known issues

> - When the capture web server is started it is possible to provide a context such as @PV or @IV and it will filter all available test scenarios to only pick out the manual ones for those tags
> - This only works on test scenarios of the BDD type, so `.feature` files that is
> - At web server start it will look for `.feature` files in and in all subdirectories of the directory named `features`
> - When saving a test result and continuing, the test result is stored in a directory named `output` on the Github runner agent, and a copy of the test result file is sent to the client browser for download

> - **Known rare issue:** Sometimes (very rarely) pressing "Save and continue" will result in a **502** but it can be fixed by reloading and choosing resubmit the page (no data is lost, it is a rare sync issue)

</br>

## Features

- ### Main screen

![image](https://github.com/user-attachments/assets/654d616e-910b-4d3c-b3ab-a2be8159da7e)

</br>

- ### Add objective evidence as images by either uploading or pasting from clipboard

![image](https://github.com/user-attachments/assets/842b3b6a-9c94-4810-998e-e68c563db74d)

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

![image](https://github.com/user-attachments/assets/3a41e613-2923-4164-86cb-9e260ae14a38)

- Each file is timestamped with time of completing the test
- Each file is labelled with the test type (`iv`, `pv`, `piv`, `ppv`) and environment (e.g. `validation`/`production`) it belongs to

![image](https://github.com/user-attachments/assets/0efb2517-bc09-4543-96c9-43a28cbc5ca7)


</br>
</br>
