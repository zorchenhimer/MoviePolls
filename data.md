# Data

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

### Choices

- ID
- Movie ID
- Cycle ID

### Users

- ID
- Name
- OAuth Token
- Email

### Votes

- Cycle ID
- Choice ID
- User ID
