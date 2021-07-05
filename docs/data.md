# Data

A bunch of this will be stored in-memory with a backup in the database.  For
example, the current cycle ID and its selections should be in-memory and all
user and past movie data should be retrieved from the database when needed.

## Things to store

- Past movies watched
- Movie details
- Users
- Admin/Mod users
- Votes
- Cycle planned end

## Tables

### Cycles

- ID
- Date planned end date
- Date actual end date

`Date ending` is just a suggestion.  Cycles must be manually reset by an admin
or mod.

### Movies

- ID
- Name
- Links
- Short description
- Cycle Added
- Cycle Watched
- Removed
- Approved

If there is cover art for a Movie, it should be stored in a folder on the
server and use the above ID in its name.  The format should either be something
like `cover-###.jpg` or simply `###.jpg`.

If `Cycle Added` is not the current cycle, and the movie has zero votes, do not
carry it over to the next cycle.

`Removed` is set when a Movie is removed by an admin or mod and it cannot be
re-added unless added by an admin or mod.

If the setting requiring movies to be approved is set, `Approved` is required
to be set before it can appear in a cycle.

### Users

- ID
- Name
- Email
- Notify on cycle end
- Notify on voted selection (if the selection at the end of a cycle is one the
  user had voted on).
- Privilege level (user/mod/admin)

### Votes

Defines a user's vote for a cycle.

- Cycle ID
- Choice ID
- User ID

### Settings and Configuration

- Key (unique string)
- Value

#### Data stored

- Max vote points (number of votes each user gets)
- Voting open (boolean)
- New choices require approval
