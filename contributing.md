# How to Contribute and development guidelines
If you want to get involved in this project you have two general ways to do so.

## Creating Issues
The first (and easiest) way to contribute is by opening issues for bugs you encountered or new features/enhancements you would like to see implemented.

Your issue should contain these elements:
- a short and clear description of the issue as title (i.e. "Add the ability to downvote movies")
- a longer explanation why you would like that feature or how the bug occurred (if applicable with some screenshots)

Some nice to haves for us:
- an idea how to implement this feature maybe with some references to other projects

## Contributing to the development
The second way to contribute is to implement/fix specific issues yourself and posting a pull request to this project.

### Getting started with development
To get your development environment started make sure to have golang installed and up to date.
After forking this repository and cloning your fork change into the `MoviePolls` folder and you will find the `Makefile` of this project.
To build the project just execute the `Makefile` with `make`. A new folder `bin` will be created with an executable file called `server`.

Before executing the resulting file you have to create the folder `MoviePolls/db` and within this new folder an empty json file `data.json` (the file is not completely empty but contains `{}`). If you do not create that file beforehand the server will not start.

After creating the necessary file and starting the server you will receive instructions how to claim admin rights on the console.
To claim admin priviledges you first have to create an account via the Login page. After your account is created go to the page posted in the console. Replace <host> with your hostname (most likely `localhost` and the configured port `:8090`) and enter the password.

### Posting the Pullrequest
After you implemented your changes in your repository (and verified that everything is still working as it should) you can post a pull request on the original repository.
Your PR should contain the following information:
- a clear title which summarizes your changes (optionally with the corresponding issue number)
- a description which explains what was done
- if you add an external dependency (i.e. a non standard library) please explain why it is used/necessary
- closing keywords to autoclose issues when merged


