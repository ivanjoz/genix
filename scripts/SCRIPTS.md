# Scripts

This document outlines how to add new operational scripts to this project.

## Convention

To maintain consistency and ease of use, all scripts are managed through a central dispatcher and a wrapper shell script.

-   `scripts/main.go`: A Go program that acts as a dispatcher to different script functionalities.
-   `app.sh`: A shell script in the root directory that provides a user-friendly interface to execute the scripts.

## Adding a New Script

When you create a new script, you **must** integrate it by following these steps:

1.  **Create the Script Logic:**
    -   Place your new Go code in a file within the `scripts/` directory (e.g., `scripts/my_new_script.go`).
    -   The script's logic must be contained within a function (e.g., `func MyNewScript(args []string)`) that can be called from the dispatcher. The file should be part of the `main` package.

2.  **Update the Go Dispatcher (`scripts/main.go`):**
    -   Add a new `case` to the `switch` statement.
    -   This case should match the desired command name for your script.
    -   Call your script's main function from this case, passing any necessary arguments (e.g., `MyNewScript(os.Args[2:])`).

3.  **Update the Shell Wrapper (`app.sh`):**
    -   Add a new `case` to the `case` statement in `app.sh`.
    -   This case should match the command name you chose in `main.go`.
    -   The command should execute the Go dispatcher and pass along any additional arguments from the command line:
        ```bash
        "my_new_script")
          (cd scripts && go run . my_new_script "${@:2}")
          ;;
        ```

4.  **Update Documentation:**
    -   Add a corresponding `MY_NEW_SCRIPT.md` file in the `scripts/` directory to document its purpose and usage.

By following these steps, you ensure that your script is available through the standard `./app.sh <script_name>` interface, making it discoverable and usable by other developers and agents.
