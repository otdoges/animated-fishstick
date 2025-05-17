/**
 * Supabase-Go JavaScript Client
 * A lightweight client for interacting with the Supabase-Go backend
 */

class SupabaseGoClient {
  /**
   * Initialize a new Supabase-Go client
   * @param {string} url - The URL of your Supabase-Go backend
   * @param {string} apiKey - The API key for authentication
   * @param {Object} options - Additional configuration options
   */
  constructor(url, apiKey, options = {}) {
    this.url = url.endsWith('/') ? url.slice(0, -1) : url;
    this.apiKey = apiKey;
    this.options = {
      autoRefreshToken: true,
      persistSession: true,
      ...options,
    };
    
    // Initialize auth
    this.auth = this._initAuth();
    
    // Initialize the query builder
    this._queryBuilder = {};
  }

  /**
   * Initialize the auth module
   * @private
   */
  _initAuth() {
    return {
      /**
       * Sign in with Clerk token
       * @param {string} token - Clerk authentication token
       */
      signInWithClerk: async (token) => {
        if (!token) {
          throw new Error('Token is required');
        }
        
        localStorage.setItem('supabase-go-token', token);
        
        return {
          user: await this._fetchUser(token),
          session: { access_token: token },
        };
      },
      
      /**
       * Sign out the current user
       */
      signOut: async () => {
        localStorage.removeItem('supabase-go-token');
        return { error: null };
      },
      
      /**
       * Get the current session
       */
      getSession: () => {
        const token = localStorage.getItem('supabase-go-token');
        return token ? { access_token: token } : null;
      },
      
      /**
       * Get the current user
       */
      getUser: async () => {
        const token = localStorage.getItem('supabase-go-token');
        if (!token) return { user: null, error: new Error('No session') };
        
        try {
          const user = await this._fetchUser(token);
          return { user, error: null };
        } catch (error) {
          return { user: null, error };
        }
      },
    };
  }
  
  /**
   * Fetch the current user profile
   * @private
   */
  async _fetchUser(token) {
    const response = await fetch(`${this.url}/api/auth/user`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch user');
    }
    
    return response.json();
  }

  /**
   * Create a query builder for a specific table
   * @param {string} tableName - The name of the table to query
   * @returns {Object} - Query builder for the specified table
   */
  from(tableName) {
    if (!this._queryBuilder[tableName]) {
      this._queryBuilder[tableName] = new QueryBuilder(this, tableName);
    }
    
    return this._queryBuilder[tableName];
  }
  
  /**
   * Execute a raw SQL query
   * @param {string} sql - The SQL query to execute
   * @param {Array} params - Query parameters
   * @returns {Promise<Object>} - Query results
   */
  async query(sql, params = []) {
    const token = localStorage.getItem('supabase-go-token');
    
    const response = await fetch(`${this.url}/api/query`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': token ? `Bearer ${token}` : '',
      },
      body: JSON.stringify({
        sql,
        parameters: params,
      }),
    });
    
    const result = await response.json();
    
    if (!response.ok) {
      throw new Error(result.error || 'Query failed');
    }
    
    return result;
  }
  
  /**
   * Get database schema
   * @returns {Promise<Object>} - Database schema
   */
  async getSchema() {
    const token = localStorage.getItem('supabase-go-token');
    
    const response = await fetch(`${this.url}/api/schema`, {
      headers: {
        'Authorization': token ? `Bearer ${token}` : '',
      },
    });
    
    const result = await response.json();
    
    if (!response.ok) {
      throw new Error(result.error || 'Failed to get schema');
    }
    
    return result;
  }
}

/**
 * QueryBuilder for constructing database queries
 */
class QueryBuilder {
  /**
   * Initialize a new QueryBuilder
   * @param {SupabaseGoClient} client - The Supabase-Go client
   * @param {string} tableName - The name of the table
   */
  constructor(client, tableName) {
    this.client = client;
    this.tableName = tableName;
    this.url = client.url;
    this.queryParams = new URLSearchParams();
    this.headers = {};
    this.method = 'GET';
    this.body = null;
    this.path = `/api/tables/${tableName}/rows`;
  }

  /**
   * Reset the query builder
   * @private
   */
  _reset() {
    this.queryParams = new URLSearchParams();
    this.headers = {};
    this.method = 'GET';
    this.body = null;
    this.path = `/api/tables/${this.tableName}/rows`;
  }

  /**
   * Execute the query
   * @private
   */
  async _execute() {
    const token = localStorage.getItem('supabase-go-token');
    
    // Add headers
    const headers = {
      ...this.headers,
      'Authorization': token ? `Bearer ${token}` : '',
    };
    
    // Add Content-Type for POST, PATCH methods
    if (['POST', 'PATCH'].includes(this.method)) {
      headers['Content-Type'] = 'application/json';
    }
    
    // Construct URL
    const queryString = this.queryParams.toString();
    const url = `${this.url}${this.path}${queryString ? `?${queryString}` : ''}`;
    
    // Prepare request
    const options = {
      method: this.method,
      headers,
    };
    
    // Add body for POST, PATCH methods
    if (['POST', 'PATCH'].includes(this.method) && this.body) {
      options.body = JSON.stringify(this.body);
    }
    
    // Execute request
    const response = await fetch(url, options);
    const result = await response.json();
    
    // Reset query builder
    this._reset();
    
    // Handle errors
    if (!response.ok) {
      throw new Error(result.error || 'Query failed');
    }
    
    return result;
  }

  /**
   * Select specific columns
   * @param {string|Array} columns - Columns to select
   * @returns {QueryBuilder} - The query builder instance
   */
  select(columns) {
    if (Array.isArray(columns)) {
      this.queryParams.set('select', columns.join(','));
    } else if (columns && columns !== '*') {
      this.queryParams.set('select', columns);
    }
    
    return this;
  }

  /**
   * Filter records with an equality filter
   * @param {string} column - Column name
   * @param {*} value - Filter value
   * @returns {QueryBuilder} - The query builder instance
   */
  eq(column, value) {
    this.queryParams.set(`${column}.eq`, value);
    return this;
  }

  /**
   * Filter records with a not-equal filter
   * @param {string} column - Column name
   * @param {*} value - Filter value
   * @returns {QueryBuilder} - The query builder instance
   */
  neq(column, value) {
    this.queryParams.set(`${column}.neq`, value);
    return this;
  }

  /**
   * Filter records with a greater-than filter
   * @param {string} column - Column name
   * @param {*} value - Filter value
   * @returns {QueryBuilder} - The query builder instance
   */
  gt(column, value) {
    this.queryParams.set(`${column}.gt`, value);
    return this;
  }

  /**
   * Filter records with a greater-than-or-equal filter
   * @param {string} column - Column name
   * @param {*} value - Filter value
   * @returns {QueryBuilder} - The query builder instance
   */
  gte(column, value) {
    this.queryParams.set(`${column}.gte`, value);
    return this;
  }

  /**
   * Filter records with a less-than filter
   * @param {string} column - Column name
   * @param {*} value - Filter value
   * @returns {QueryBuilder} - The query builder instance
   */
  lt(column, value) {
    this.queryParams.set(`${column}.lt`, value);
    return this;
  }

  /**
   * Filter records with a less-than-or-equal filter
   * @param {string} column - Column name
   * @param {*} value - Filter value
   * @returns {QueryBuilder} - The query builder instance
   */
  lte(column, value) {
    this.queryParams.set(`${column}.lte`, value);
    return this;
  }

  /**
   * Filter records with a LIKE filter
   * @param {string} column - Column name
   * @param {string} pattern - LIKE pattern
   * @returns {QueryBuilder} - The query builder instance
   */
  like(column, pattern) {
    this.queryParams.set(`${column}.like`, pattern);
    return this;
  }

  /**
   * Filter records with an IN filter
   * @param {string} column - Column name
   * @param {Array} values - Array of values
   * @returns {QueryBuilder} - The query builder instance
   */
  in(column, values) {
    if (!Array.isArray(values)) {
      throw new Error('Values must be an array');
    }
    
    this.queryParams.set(`${column}.in`, values.join(','));
    return this;
  }

  /**
   * Order results by a column
   * @param {string} column - Column name
   * @param {string} direction - Sort direction ('asc' or 'desc')
   * @returns {QueryBuilder} - The query builder instance
   */
  order(column, direction = 'asc') {
    if (!['asc', 'desc'].includes(direction.toLowerCase())) {
      throw new Error('Order direction must be "asc" or "desc"');
    }
    
    this.queryParams.set('order_by', column);
    this.queryParams.set('order_dir', direction.toLowerCase());
    return this;
  }

  /**
   * Limit the number of rows returned
   * @param {number} count - Maximum number of rows
   * @returns {QueryBuilder} - The query builder instance
   */
  limit(count) {
    this.queryParams.set('page_size', count);
    return this;
  }

  /**
   * Set the page for pagination
   * @param {number} number - Page number (1-based)
   * @returns {QueryBuilder} - The query builder instance
   */
  page(number) {
    this.queryParams.set('page', number);
    return this;
  }

  /**
   * Execute a SELECT query
   * @returns {Promise<Object>} - Query results
   */
  async get() {
    this.method = 'GET';
    return this._execute();
  }

  /**
   * Insert a new row
   * @param {Object} data - Row data to insert
   * @returns {Promise<Object>} - Inserted row
   */
  async insert(data) {
    this.method = 'POST';
    this.body = data;
    this.path = `/api/tables/${this.tableName}`;
    return this._execute();
  }

  /**
   * Update a row by ID
   * @param {string|number} id - Row ID
   * @param {Object} data - Row data to update
   * @returns {Promise<Object>} - Updated row
   */
  async update(id, data) {
    this.method = 'PATCH';
    this.body = data;
    this.path = `/api/tables/${this.tableName}/rows/${id}`;
    return this._execute();
  }

  /**
   * Delete a row by ID
   * @param {string|number} id - Row ID
   * @returns {Promise<Object>} - Deleted row
   */
  async delete(id) {
    this.method = 'DELETE';
    this.path = `/api/tables/${this.tableName}/rows/${id}`;
    return this._execute();
  }

  /**
   * Get a row by ID
   * @param {string|number} id - Row ID
   * @returns {Promise<Object>} - Row data
   */
  async getById(id) {
    this.method = 'GET';
    this.path = `/api/tables/${this.tableName}/rows/${id}`;
    return this._execute();
  }
}

/**
 * Create a new Supabase-Go client
 * @param {string} url - The URL of your Supabase-Go backend
 * @param {string} apiKey - The API key for authentication
 * @param {Object} options - Additional configuration options
 * @returns {SupabaseGoClient} - A new Supabase-Go client
 */
function createClient(url, apiKey, options = {}) {
  return new SupabaseGoClient(url, apiKey, options);
}

// Export the library
export { createClient };
export default { createClient };
