# Overview

## Purpose

A more democratic process for selecting a movie to watch, allowing users to
suggest and vote on movies to watch.

## Specific features

Note: not all of this stuff is implemented yet.

- Twitch OAuth to verify users
- Running totals (see below)
- (optional) Email reminders on movies voted for
- Links to movie details (IMDB, AniDB, MAL, etc.)
- Info on the upcoming MovieNight (if a planned end date is set)
- Link to MovieNight stream if live?
- Movie history (selected movies and dates watched)
- Votes decay (eg., only last for a given number of cycles)

## In-depth Explanation

Users are given a fixed number of vote points.  One vote = one point.  Each
user is allowed to only vote once per movie, regardless of the number of
available points.  Users can re-distribute or undo their votes during an
active cycle.

Once a movie is chosen, that movie is added to a "watched/chosen" list and
cannot be re-added (admin overwritable?).  Users that had voted on the selected
movie will get their vote points back that can be used for the next movie.

Movies can only be removed if they have zero votes after a cycle, if they are
chosen, or if an admin or mod removes them.  Movies that are removed after zero
votes can be re-added at a later cycle, movies removed by an admin or mod
cannot.

A cycle is defined as the time between two movie nights.  Typically one or two
weeks.  A cycle is reset by an admin or a mod.  Resetting a cycle chooses a
movie and the process described above starts.  The admin or mod that resets the
cycle can define the number of movies to choose for that cycle.

Once a cycle is reset, notifications are sent out to users that have opted into
receiving notifications.  Notifications *WILL NOT* be an opt-out but instead an
opt-in process.  Users should not receive notifications if they do not
explicitly approve beforehand.  Removing approval for notifications should be a
single-click process.

Movies can be suggested at any time by a user.  Once a movie is added it is
immediately available for voting.  The exception to this is if the server is
configured to require admin or mod approval for each suggestion.  Movies can
only be added if they have not been previously removed by an admin or mod.

Adding a movie must include the name and at least one link for more
information.  The link should be to either IMDB, AniDB, or MAL.  Possibly
auto-fetch info for a short synopsis and a cover image.

A vote for a selection will decay after a configurable number of cycles
(default two?).  A decayed vote will be removed from the selection it was
assigned and re-add a vote point to the user that cast the vote.

## Data Backends

- MySQL (default b/c I can offload it on my hosting)
- PostgreSQL
- Flat file JSON (meant mainly for developing and debugging)

## Mod/Admin differences

Mod and Admin abilities:
- Approve/Deny pending entries
- Ban/Unban users
- Re-add existing/duplicate entry
- End cycle
- Ignore rate limit
- Trigger cycle notifications

Admin only:
- Add new user
- Edit existing user (eg, change password, change privilage level)
- Change server configuraton settings
- Dedicated login at /admin/login (available even when the simple login method is disabled)
- Test notifications

## How to Contribute and development guidelines
If you want to get involved in this project you have two general ways to do so.
### Creating Issues
The first (and easiest) way to contribute is by opening issues for bugs you encountered or new features/enhancements you would like to see implemented.

Your issue should contain these elements:
- a short and clear description of the issue as title (i.e. "Add the ability to downvote movies")
- a longer explanation why you would like that feature or how the bug occurred (if applicable with some screenshots)

Some nice to haves for us:
- an idea how to implement this feature maybe with some references to other projects

### Contributing to the development
The second way to contribute is to implement/fix specific issues yourself and posting a pull request to this project.

#### Getting started with development
To get your development environment started make sure to have golang installed and up to date.
After forking this repository and cloning your fork change into the `MoviePolls` folder and you will find the `Makefile` of this project.
To build the project just execute the `Makefile` with `make`. A new folder `bin` will be created with an executable file called `server`.

Before executing the resulting file you have to create the folder `MoviePolls/db` and within this new folder an empty json file `data.json` (the file is not completely empty but contains `{}`). If you do not create that file beforehand the server will not start.

After creating the necessary file and starting the server you will receive instructions how to claim admin rights on the console.
To claim admin priviledges you first have to create an account via the Login page. After your account is created go to the page posted in the console. Replace <host> with your hostname (most likely `localhost` and the configured port `:8090`) and enter the password.

#### Posting the Pullrequest
After you implemented your changes in your repository (and verified that everything is still working as it should) you can post a pull request on the original repository.
Your PR should contain the following information:
- a clear title which summarizes your changes (optionally with the corresponding issue number)
- a description which explains what was done
- if you add an external dependency (i.e. a non standard library) please explain why it is used/necessary
- closing keywords to autoclose issues when merged


# License

This project is licensed under the MIT license.  See `license.md` for details.
