import { BrowserRouter as Router } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { AuthProvider } from './contexts/AuthContext';
import { ThemeProvider } from './contexts/ThemeContext';
import AppRoutes from './routes';

function App() {
  return (
    <Router>
      <ThemeProvider>
        <AuthProvider>
              <AppRoutes />
              <Toaster 
                position="top-right"
                toastOptions={{
                  style: {
                    background: '#171717',
                    color: '#fff',
                    border: '1px solid #262626'
                  },
                }}
              />
        </AuthProvider>
      </ThemeProvider>
    </Router>
  );
}

export default App;