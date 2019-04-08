# Data

A bunch of this will be stored in-memory with a backup in the database.  For
example, the current cycle ID and its selections should be in-memory and all
user and past movie data should be retrieved from the database when needed.

## Things to store

- Past movies selected
- Current movie selection
- Movie details
- Users
- Admin/Mod users
- Current Votes
- Cycle start/end

## Tables

### Cycles

- ID
- Date started
- Predicted date end

### Movies

- ID
- Name
- Links
- Short description
- Cycle Added

If there is cover art for a Movie, it should be stored in a folder on the
server and use the above ID in its name.  The format should either be something
like `cover-###.jpg` or simply `###.jpg`.

If `Cycle Added` is not the current cycle, and the movie has zero votes, do not
carry it over to the next cycle.

### Choices

Defines the available choices for a cycle.  Choices without votes that were
added before the current cycle are removed at the end of the current cycle.

- ID
- Movie ID
- Cycle ID

### Users

- ID
- Name
- OAuth Token
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

- Current cycle ID (only once cycle is enabled at a time)
- Default vote points (number of votes each user gets)
- Voting open (boolean)
- (Other misc settings)
