// Android emulator defaults to the host machine. Set a LAN URL on a real device.
export const API_BASE_URL = (process.env.EXPO_PUBLIC_API_URL || 'http://10.0.2.2:8080').replace(/\/$/, '');
