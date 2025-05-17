# Supabase Clone in Go

A high-performance Supabase clone built with Go, designed for serverless deployment on platforms like Vercel and Netlify. This project provides similar functionality to Supabase, including database operations, authentication with Clerk, and row-level security policies.

## Features

- **PostgreSQL Database**: Full support for PostgreSQL with automatic API generation
- **RESTful API**: Automatically generated API endpoints for your database tables
- **Row Level Security (RLS)**: Secure data access with database-level policies
- **Authentication**: Seamless integration with Clerk for authentication
- **Client SDK**: JavaScript/TypeScript client with React hooks
- **Serverless Ready**: Deploy on Vercel, Netlify, or any serverless platform
- **High Performance**: Written in Go for optimal performance

## Architecture

The project consists of several components:

1. **Go Backend**: Core API server with database connectivity
2. **JavaScript SDK**: Client-side library for easy integration
3. **React Hooks**: Simplified data access in React applications
4. **PostgreSQL Database**: Storage layer with security policies

## Getting Started

### Prerequisites

- Go 1.18 or later
- PostgreSQL 12 or later
- Node.js 14 or later (for frontend)
- Docker and Docker Compose (for local development)

### Local Development

1. Clone the repository:

```bash
git clone https://github.com/yourusername/supabase-in-go
cd supabase-in-go
```

2. Create a `.env` file in the root directory:

```
CLERK_PUBLISHABLE_KEY=your_clerk_publishable_key
CLERK_SECRET_KEY=your_clerk_secret_key
```

3. Start the development environment:

```bash
docker-compose up
```

This will start:
- PostgreSQL database
- Go backend API
- Next.js frontend

4. Access your development environment:
   - Frontend: http://localhost:3000
   - API: http://localhost:8080

### Environment Variables

#### Backend

| Variable | Description | Default |
|----------|-------------|---------|
| DB_HOST | Database host | localhost |
| DB_PORT | Database port | 5432 |
| DB_USER | Database user | postgres |
| DB_PASSWORD | Database password | postgres |
| DB_NAME | Database name | supabase |
| DB_SSL_MODE | Database SSL mode | disable |
| PORT | API server port | 8080 |
| CLERK_PUBLISHABLE_KEY | Clerk publishable key | |
| CLERK_SECRET_KEY | Clerk secret key | |
| CORS_ALLOW_ORIGINS | CORS allowed origins | * |

#### Frontend

| Variable | Description |
|----------|-------------|
| NEXT_PUBLIC_API_URL | URL of the backend API |
| NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY | Clerk publishable key |

## Deployment

### Vercel

1. Set environment variables in Vercel project settings
2. Deploy the backend:

```bash
cd backend
vercel
```

3. Deploy the frontend:

```bash
cd front-end
vercel
```

### Netlify

1. Set environment variables in Netlify project settings
2. Deploy the backend:

```bash
cd backend
netlify deploy
```

3. Deploy the frontend:

```bash
cd front-end
netlify deploy
```

## Client SDK Usage

### JavaScript/TypeScript

```javascript
import { createClient } from 'supabase-go-client';

// Initialize client
const supabase = createClient('https://your-api-url.com', 'your-api-key');

// Fetch data
const { data, error } = await supabase
  .from('users')
  .select('id, name, email')
  .eq('active', true)
  .limit(10);

// Insert data
const { data, error } = await supabase
  .from('users')
  .insert({ name: 'John Doe', email: 'john@example.com' });

// Update data
const { data, error } = await supabase
  .from('users')
  .update({ name: 'Jane Doe' })
  .eq('id', 1);

// Delete data
const { data, error } = await supabase
  .from('users')
  .delete()
  .eq('id', 1);
```

### React Hooks

```jsx
import { SupabaseGoProvider, useQuery, useMutation } from 'supabase-go-client/react';
import { createClient } from 'supabase-go-client';

const supabase = createClient('https://your-api-url.com', 'your-api-key');

function App() {
  return (
    <SupabaseGoProvider client={supabase}>
      <UserList />
    </SupabaseGoProvider>
  );
}

function UserList() {
  // Query data
  const { data, error, isLoading } = useQuery('users', {
    columns: 'id, name, email',
    filters: { active: true },
    limit: 10,
  });

  // Mutation
  const { mutate, isLoading: isMutating } = useMutation('users', 'insert');

  const handleAddUser = async () => {
    await mutate({ name: 'New User', email: 'new@example.com' });
  };

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p>Error: {error}</p>;

  return (
    <div>
      <button onClick={handleAddUser} disabled={isMutating}>
        Add User
      </button>
      <ul>
        {data.map(user => (
          <li key={user.id}>{user.name}</li>
        ))}
      </ul>
    </div>
  );
}
```

## Row Level Security

RLS policies can be managed via the API or directly in the database:

```sql
-- Enable RLS on a table
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Create a policy that allows users to see only their own data
CREATE POLICY user_see_own_data ON users
  FOR SELECT
  USING (auth.uid() = user_id);
```

## API Reference

### Authentication

| Endpoint | Method | Description |
|----------|--------|-------------|
| /api/auth/user | GET | Get current user information |

### Database

| Endpoint | Method | Description |
|----------|--------|-------------|
| /api/tables | GET | List all tables |
| /api/tables/:table | GET | Get table information |
| /api/tables/:table/columns | GET | Get table columns |
| /api/tables/:table/rows | GET | Query table rows |
| /api/tables/:table | POST | Insert new row |
| /api/tables/:table/rows/:id | GET | Get row by ID |
| /api/tables/:table/rows/:id | PATCH | Update row by ID |
| /api/tables/:table/rows/:id | DELETE | Delete row by ID |

### Schema

| Endpoint | Method | Description |
|----------|--------|-------------|
| /api/schema | GET | Get database schema |

### RLS Policies

| Endpoint | Method | Description |
|----------|--------|-------------|
| /api/rls/policies | GET | List all RLS policies |
| /api/rls/policies | POST | Create new RLS policy |
| /api/rls/policies/:id | GET | Get RLS policy by ID |
| /api/rls/policies/:id | PATCH | Update RLS policy by ID |
| /api/rls/policies/:id | DELETE | Delete RLS policy by ID |

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
