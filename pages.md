# Pages

## Main / Landing

This will be the default landing page.  It should show the currently running
poll with an option to vote.  If the user is not auth'd, instead of allowing
the user to vote, ask for a login.  The current poll should be shown
regardless.

## Movie details

Details of a given movie.  This should include:

- Movie name
- Info links (IMDB, AniDB, MAL, etc.)
- Poster
- Short description
- If watched, when
- If not watched, number of votes (if any)
- If not watched (eg, in current cycle) list of user that voted for movie
- User that added the movie

## Cycle list

Details of past cycles.  This should include:

- Movie name
- Link to movie info page (described above)
- Votes for movie
- Date watched
- Cycle start and end dates

## Login

This page will allow a user to login and auth their account.  The main method
for this will be Twitch's OAuth.  Prompt the user for access to their display
name on twitch.

If the user is logging in for the first time, prompt for reminder
notifications.  If they do not want any notifications, do not get their email
via OAuth.  If they do want notifictions, ask for permission to read it via
OAuth.

*If the user doesn't want notifications, I don't want their email.*

## Add Movie

Only allow logged-in users to add movies.  The following details should be
prompted for on this page:

- Movie name
- Info links (IMDB, AniDB, MAL, etc.)
- Short description
