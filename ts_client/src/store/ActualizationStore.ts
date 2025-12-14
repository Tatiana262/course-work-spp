import { makeAutoObservable } from "mobx";

export default class ActualizationStore {
    // ID объектов, которые сейчас обновляются
    private _processingIds: Set<string> = new Set();

    public updates: Map<string, number> = new Map();

    public version: number = 0;

    constructor() {
        makeAutoObservable(this);
    }

    add(id: string) {
        this._processingIds.add(id);
    }

    remove(id: string) {
        this._processingIds.delete(id);
    }

    isProcessing(id: string) {
        return this._processingIds.has(id);
    }

    // Вызываем, когда SSE прислал "completed"
    markAsUpdated(id: string) {
        this.updates.set(id, Date.now());
        this.version += 1;
    }
}