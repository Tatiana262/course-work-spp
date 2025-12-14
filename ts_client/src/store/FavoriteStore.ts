import { makeAutoObservable } from "mobx";

export default class FavoriteStore {
    // ID как Set для быстрого поиска .has()
    private _favoritesIds: Set<string> = new Set();

    constructor() {
        makeAutoObservable(this);
    }

    // Записать массив ID (при загрузке приложения)
    setFavorites(ids: string[]) {
        this._favoritesIds = new Set(ids);
    }

    // Добавить один (при клике лайка)
    addFavorite(id: string) {
        this._favoritesIds.add(id);
    }

    // Удалить один (при клике дизлайка)
    removeFavorite(id: string) {
        this._favoritesIds.delete(id);
    }

    // Проверка: лайкнут ли объект?
    isFavorite(id: string): boolean {
        return this._favoritesIds.has(id);
    }

    // Очистка (при выходе)
    clear() {
        this._favoritesIds.clear();
    }
}