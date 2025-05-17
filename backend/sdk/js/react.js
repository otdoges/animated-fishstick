import { useEffect, useState, useContext, createContext } from 'react';

// Create a context for the Supabase client
const SupabaseGoContext = createContext(null);

/**
 * Provider component for Supabase-Go client
 * @param {Object} props - Component props
 * @param {Object} props.client - Supabase-Go client instance
 * @param {Object} props.children - Child components
 */
export function SupabaseGoProvider({ client, children }) {
  return (
    <SupabaseGoContext.Provider value={client}>
      {children}
    </SupabaseGoContext.Provider>
  );
}

/**
 * Hook to access the Supabase-Go client
 * @returns {Object} Supabase-Go client
 */
export function useSupabaseGo() {
  const client = useContext(SupabaseGoContext);
  
  if (!client) {
    throw new Error('useSupabaseGo must be used within a SupabaseGoProvider');
  }
  
  return client;
}

/**
 * Hook for querying data from a table
 * @param {string} tableName - The name of the table to query
 * @param {Object} queryOptions - Query options
 * @param {string|Array} queryOptions.columns - Columns to select
 * @param {Object} queryOptions.filters - Filter conditions
 * @param {string} queryOptions.orderBy - Column to order by
 * @param {string} queryOptions.orderDirection - Order direction ('asc' or 'desc')
 * @param {number} queryOptions.limit - Maximum number of rows
 * @param {number} queryOptions.page - Page number
 * @param {Array} dependencies - Array of dependencies to trigger refetch
 * @returns {Object} Query results and state
 */
export function useQuery(tableName, queryOptions = {}, dependencies = []) {
  const client = useSupabaseGo();
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0,
    totalPages: 0,
  });

  useEffect(() => {
    const fetchData = async () => {
      try {
        setIsLoading(true);
        
        // Start building the query
        let query = client.from(tableName);
        
        // Apply columns
        if (queryOptions.columns) {
          query = query.select(queryOptions.columns);
        }
        
        // Apply filters
        if (queryOptions.filters) {
          Object.entries(queryOptions.filters).forEach(([filter, value]) => {
            const [column, operator = 'eq'] = filter.split('.');
            
            switch (operator) {
              case 'eq':
                query = query.eq(column, value);
                break;
              case 'neq':
                query = query.neq(column, value);
                break;
              case 'gt':
                query = query.gt(column, value);
                break;
              case 'gte':
                query = query.gte(column, value);
                break;
              case 'lt':
                query = query.lt(column, value);
                break;
              case 'lte':
                query = query.lte(column, value);
                break;
              case 'like':
                query = query.like(column, value);
                break;
              case 'in':
                query = query.in(column, value);
                break;
              default:
                query = query.eq(column, value);
            }
          });
        }
        
        // Apply ordering
        if (queryOptions.orderBy) {
          query = query.order(
            queryOptions.orderBy,
            queryOptions.orderDirection || 'asc'
          );
        }
        
        // Apply pagination
        if (queryOptions.limit) {
          query = query.limit(queryOptions.limit);
        }
        
        if (queryOptions.page) {
          query = query.page(queryOptions.page);
        }
        
        // Execute the query
        const result = await query.get();
        
        // Set the data
        setData(result.data || []);
        
        // Set pagination info
        setPagination({
          page: result.page || 1,
          pageSize: result.page_size || 10,
          total: result.total || 0,
          totalPages: result.total_pages || 0,
        });
        
        setError(null);
      } catch (err) {
        setError(err.message || 'An error occurred');
        setData(null);
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, [client, tableName, ...dependencies]);

  return {
    data,
    error,
    isLoading,
    pagination,
    refetch: () => {
      setIsLoading(true);
      // Force a re-render to trigger the useEffect
      setData(null);
    },
  };
}

/**
 * Hook for fetching a single row by ID
 * @param {string} tableName - The name of the table
 * @param {string|number} id - Row ID
 * @param {Object} options - Query options
 * @param {string|Array} options.columns - Columns to select
 * @param {Array} dependencies - Array of dependencies to trigger refetch
 * @returns {Object} Query results and state
 */
export function useRow(tableName, id, options = {}, dependencies = []) {
  const client = useSupabaseGo();
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      if (!id) {
        setData(null);
        setIsLoading(false);
        return;
      }
      
      try {
        setIsLoading(true);
        
        // Start building the query
        let query = client.from(tableName);
        
        // Apply columns
        if (options.columns) {
          query = query.select(options.columns);
        }
        
        // Get the row by ID
        const result = await query.getById(id);
        
        // Set the data
        setData(result);
        setError(null);
      } catch (err) {
        setError(err.message || 'An error occurred');
        setData(null);
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, [client, tableName, id, ...dependencies]);

  return {
    data,
    error,
    isLoading,
    refetch: () => {
      setIsLoading(true);
      // Force a re-render to trigger the useEffect
      setData(null);
    },
  };
}

/**
 * Hook for mutations (insert, update, delete)
 * @param {string} tableName - The name of the table
 * @param {string} type - Mutation type ('insert', 'update', 'delete')
 * @returns {Object} Mutation function and state
 */
export function useMutation(tableName, type) {
  const client = useSupabaseGo();
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(false);

  const mutate = async (payload, id = null) => {
    try {
      setIsLoading(true);
      
      let result;
      
      switch (type) {
        case 'insert':
          result = await client.from(tableName).insert(payload);
          break;
        case 'update':
          if (!id) {
            throw new Error('ID is required for update operations');
          }
          result = await client.from(tableName).update(id, payload);
          break;
        case 'delete':
          if (!id) {
            throw new Error('ID is required for delete operations');
          }
          result = await client.from(tableName).delete(id);
          break;
        default:
          throw new Error(`Invalid mutation type: ${type}`);
      }
      
      setData(result);
      setError(null);
      return result;
    } catch (err) {
      setError(err.message || 'An error occurred');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  return {
    mutate,
    data,
    error,
    isLoading,
    reset: () => {
      setData(null);
      setError(null);
    },
  };
}

/**
 * Hook for authentication with Clerk
 * @returns {Object} Auth methods and state
 */
export function useAuth() {
  const client = useSupabaseGo();
  const [user, setUser] = useState(null);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const loadUser = async () => {
      try {
        setIsLoading(true);
        const { user, error } = await client.auth.getUser();
        
        if (error) {
          setError(error.message);
          setUser(null);
        } else {
          setUser(user);
          setError(null);
        }
      } catch (err) {
        setError(err.message || 'Authentication error');
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };

    loadUser();
  }, [client]);

  const signInWithClerk = async (token) => {
    try {
      setIsLoading(true);
      const { user, error } = await client.auth.signInWithClerk(token);
      
      if (error) {
        setError(error.message);
        return { error };
      }
      
      setUser(user);
      setError(null);
      return { user };
    } catch (err) {
      const error = err.message || 'Sign in failed';
      setError(error);
      return { error };
    } finally {
      setIsLoading(false);
    }
  };

  const signOut = async () => {
    try {
      setIsLoading(true);
      await client.auth.signOut();
      setUser(null);
      setError(null);
      return { error: null };
    } catch (err) {
      const error = err.message || 'Sign out failed';
      setError(error);
      return { error };
    } finally {
      setIsLoading(false);
    }
  };

  return {
    user,
    error,
    isLoading,
    signInWithClerk,
    signOut,
  };
}

/**
 * Hook for executing raw SQL queries
 * @returns {Function} Query function
 */
export function useRawQuery() {
  const client = useSupabaseGo();
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(false);

  const execute = async (sql, params = []) => {
    try {
      setIsLoading(true);
      const result = await client.query(sql, params);
      setData(result.data);
      setError(null);
      return result;
    } catch (err) {
      setError(err.message || 'Query failed');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  return {
    execute,
    data,
    error,
    isLoading,
    reset: () => {
      setData(null);
      setError(null);
    },
  };
}
