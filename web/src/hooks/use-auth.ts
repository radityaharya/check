import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  AuthState,
  LoginCredentials,
  User,
} from '@/types';

// Auth API functions
async function checkSetup(): Promise<{ needs_setup: boolean }> {
  const response = await fetch('/api/auth/setup/check');
  return response.json();
}

async function checkAuth(): Promise<{ user: User } | null> {
  const response = await fetch('/api/auth/check');
  if (!response.ok) return null;
  return response.json();
}

async function login(credentials: LoginCredentials): Promise<{ user: User }> {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credentials),
  });
  if (!response.ok) {
    throw new Error('Invalid username or password');
  }
  return response.json();
}

async function setup(credentials: LoginCredentials): Promise<{ user: User }> {
  const response = await fetch('/api/auth/setup', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credentials),
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || 'Setup failed');
  }
  return response.json();
}

async function logout(): Promise<void> {
  await fetch('/api/auth/logout', { method: 'POST' });
}

// Passkey functions
async function beginPasskeyLogin(username: string = ''): Promise<{ options: any; token: string }> {
  const response = await fetch('/api/auth/passkey/begin-login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username }),
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || 'Failed to start passkey login');
  }
  return response.json();
}

async function finishPasskeyLogin(data: any): Promise<{ user: User }> {
  const response = await fetch('/api/auth/passkey/finish-login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Passkey authentication failed');
  }
  return response.json();
}

async function beginPasskeyRegistration(): Promise<any> {
  const response = await fetch('/api/auth/passkey/begin-registration', {
    method: 'POST',
  });
  if (!response.ok) {
    throw new Error('Failed to start passkey registration');
  }
  return response.json();
}

async function finishPasskeyRegistration(data: any): Promise<void> {
  const response = await fetch('/api/auth/passkey/finish-registration', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Failed to register passkey');
  }
}

// React Query hooks
export function useAuthCheck() {
  return useQuery({
    queryKey: ['auth', 'check'],
    queryFn: async (): Promise<AuthState> => {
      const setupData = await checkSetup();
      
      if (setupData.needs_setup) {
        return {
          isAuthenticated: false,
          needsSetup: true,
          user: null,
        };
      }

      const authData = await checkAuth();
      return {
        isAuthenticated: !!authData,
        needsSetup: false,
        user: authData?.user || null,
      };
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: login,
    onSuccess: (data) => {
      queryClient.setQueryData(['auth', 'check'], {
        isAuthenticated: true,
        needsSetup: false,
        user: data.user,
      });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
      queryClient.invalidateQueries({ queryKey: ['groups'] });
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      queryClient.invalidateQueries({ queryKey: ['stats'] });
    },
  });
}

export function useSetup() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: setup,
    onSuccess: (data) => {
      queryClient.setQueryData(['auth', 'check'], {
        isAuthenticated: true,
        needsSetup: false,
        user: data.user,
      });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
      queryClient.invalidateQueries({ queryKey: ['groups'] });
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.setQueryData(['auth', 'check'], {
        isAuthenticated: false,
        needsSetup: false,
        user: null,
      });
      queryClient.clear();
    },
  });
}

export function usePasskeyLogin() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ username }: { username?: string }) => {
      const { options, token } = await beginPasskeyLogin(username);
      
      const credential = await navigator.credentials.get({
        publicKey: decodePublicKeyRequestOptions(options.publicKey),
      }) as PublicKeyCredential;
      
      if (!credential) {
        throw new Error('No credential returned');
      }
      
      const response = credential.response as AuthenticatorAssertionResponse;
      
      return finishPasskeyLogin({
        username: username || '',
        token,
        id: credential.id,
        rawId: bufferToBase64(credential.rawId),
        type: credential.type,
        response: {
          authenticatorData: bufferToBase64(response.authenticatorData),
          clientDataJSON: bufferToBase64(response.clientDataJSON),
          signature: bufferToBase64(response.signature),
          userHandle: response.userHandle ? bufferToBase64(response.userHandle) : null,
        },
      });
    },
    onSuccess: (data) => {
      queryClient.setQueryData(['auth', 'check'], {
        isAuthenticated: true,
        needsSetup: false,
        user: data.user,
      });
      queryClient.invalidateQueries({ queryKey: ['checks'] });
      queryClient.invalidateQueries({ queryKey: ['groups'] });
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

export function useRegisterPasskey() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ name }: { name: string }) => {
      const options = await beginPasskeyRegistration();
      
      const credential = await navigator.credentials.create({
        publicKey: decodePublicKeyCreationOptions(options.publicKey),
      }) as PublicKeyCredential;
      
      if (!credential) {
        throw new Error('No credential returned');
      }
      
      const response = credential.response as AuthenticatorAttestationResponse;
      
      await finishPasskeyRegistration({
        name,
        id: credential.id,
        rawId: bufferToBase64(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: bufferToBase64(response.attestationObject),
          clientDataJSON: bufferToBase64(response.clientDataJSON),
        },
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['passkeys'] });
    },
  });
}

// WebAuthn helper functions
function decodePublicKeyRequestOptions(publicKey: any): PublicKeyCredentialRequestOptions {
  const decoded = { ...publicKey };

  if (decoded.challenge) {
    decoded.challenge = base64ToBuffer(decoded.challenge);
  }

  if (decoded.allowCredentials) {
    decoded.allowCredentials = decoded.allowCredentials.map((cred: any) => ({
      ...cred,
      id: base64ToBuffer(cred.id),
    }));
  }

  return decoded;
}

function decodePublicKeyCreationOptions(publicKey: any): PublicKeyCredentialCreationOptions {
  const decoded = { ...publicKey };

  if (decoded.challenge) {
    decoded.challenge = base64ToBuffer(decoded.challenge);
  }

  if (decoded.user?.id) {
    decoded.user.id = base64ToBuffer(decoded.user.id);
  }

  if (decoded.excludeCredentials) {
    decoded.excludeCredentials = decoded.excludeCredentials.map((cred: any) => ({
      ...cred,
      id: base64ToBuffer(cred.id),
    }));
  }

  return decoded;
}

function base64ToBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64.replace(/-/g, '+').replace(/_/g, '/'));
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

function bufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}
