import { makeAutoObservable, runInAction } from 'mobx';
import { User } from '@/shared/api/types';

class UserStore {
  user: User | null = null;
  isAuthenticated: boolean = false;
  // Можно добавить isLoading, error и т.д.

  constructor() {
    makeAutoObservable(this);
  }

  // Action для установки пользователя
  setUser(user: User | null) {
    this.user = user;
    this.isAuthenticated = !!user;
  }
  
  // Пример асинхронного action, который вы бы вызвали из use case/хука
  async login(loginFn: () => Promise<User>) {
    try {
      const user = await loginFn();
      runInAction(() => {
        this.setUser(user);
      });
    } catch (error) {
      // обработка ошибки
    }
  }
}

// Экспортируем синглтон-инстанс стора
export const userStore = new UserStore();