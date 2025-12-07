import { AuthPage } from '@/pages/AuthPage';
import { Header } from '@/shared/ui/Header';
import { Routes, Route } from 'react-router-dom';
// Импортируем наши будущие страницы
// import { ObjectsListPage } from '@/pages/ObjectsListPage';
// import { AuthPage } from '@/pages/AuthPage';

export const App = () => {
  return (
    <div>
      <h1>My Real Estate App</h1>
      <Header />
      <main>
        <Routes>
          {/* <Route path="/" element={<ObjectsListPage />} /> */}
          {/* <Route path="/login" element={<AuthPage />} /> */}
          {/* 
          <Route 
            path="/favorites" 
            element={
              <RequireAuth>
                <FavoritesPage />
              </RequireAuth>
            } 
          />
          */}
          <Route path="/" element={<div>Main Page Placeholder</div>} />
          <Route path="/auth" element={<AuthPage />} />
        </Routes>
      </main>    
    </div>
  );
};