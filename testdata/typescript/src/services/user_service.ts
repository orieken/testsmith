import axios, { AxiosInstance } from 'axios';
import { createHash } from 'crypto';
import { readFileSync } from 'fs';

export interface User {
  id: string;
  email: string;
  passwordHash: string;
}

export interface CreateUserDTO {
  email: string;
  password: string;
}

export class UserService {
  private http: AxiosInstance;
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
    this.http = axios.create({ baseURL: baseUrl });
  }

  async getUser(id: string): Promise<User | null> {
    const response = await this.http.get<User>(`/users/${id}`);
    return response.data ?? null;
  }

  async createUser(dto: CreateUserDTO): Promise<User> {
    const passwordHash = hashPassword(dto.password);
    const response = await this.http.post<User>('/users', {
      email: dto.email,
      passwordHash,
    });
    return response.data;
  }

  async deleteUser(id: string): Promise<void> {
    await this.http.delete(`/users/${id}`);
  }
}

export function hashPassword(password: string): string {
  return createHash('sha256').update(password).digest('hex');
}

export function loadConfig(filePath: string): Record<string, string> {
  const raw = readFileSync(filePath, 'utf-8');
  return JSON.parse(raw);
}
