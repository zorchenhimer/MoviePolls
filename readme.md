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
