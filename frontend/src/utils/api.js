export const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');
export const NODE_URL = process.env.REACT_APP_NODE_URL || (window.location.port === '3000' ? 'http://localhost:4000' : '');
