import Admin from './pages/Admin';
import Auth from './pages/Auth';
import Favorites from './pages/Favorites';
import Filters from './pages/FiltersPage';
import Listings from './pages/Listings';
import ObjectPage from './pages/ObjectPage';

import { ADMIN_ROUTE, LOGIN_ROUTE, REGISTRATION_ROUTE, MAIN_ROUTE, LISTINGS_ROUTE, OBJECT_ROUTE, FAVORITES_ROUTE } from './utils/consts';


interface IRoute {
    path: string;
    Component: React.ComponentType; // Это тип любого React-компонента (функционального или классового)
}

export const authRoutes: IRoute[] = [
    {
        //путь - ссылка, по которой страница будет отрабатывать
        path: ADMIN_ROUTE,
        //сама страница
        Component: Admin
    },
    {
        path: FAVORITES_ROUTE,
        Component: Favorites
    },
];


export const publicRoutes: IRoute[] = [
    {
        path: LOGIN_ROUTE,
        Component: Auth
    },

    {
        path: REGISTRATION_ROUTE,
        Component: Auth
    },

    {
        path: MAIN_ROUTE,
        Component: Filters
    },
    {
        path: LISTINGS_ROUTE,
        Component: Listings
    },
    {
        path: OBJECT_ROUTE,
        Component: ObjectPage
    },
]