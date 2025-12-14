import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css'; 
import { createContext } from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import UserStore from './store/UserStore';
import FavoriteStore from './store/FavoriteStore.ts';
import ActualizationStore from './store/ActualizationStore.ts';


// Описываем, что лежит в контексте
interface State {
  user: UserStore;
  favorites: FavoriteStore;
  actualization: ActualizationStore;
}

// Создаем стор
const userStore = new UserStore();
const favoritesStore = new FavoriteStore();
const actualizationStore = new ActualizationStore();

// Контекст может быть null изначально, но мы сразу передаем значение
export const Context = createContext<State>({
  user: userStore,
  favorites: favoritesStore,
  actualization: actualizationStore,
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <Context.Provider value={{
    user: userStore,
    favorites: favoritesStore,
    actualization: actualizationStore,
  }}>
    <App />
  </Context.Provider>,
)