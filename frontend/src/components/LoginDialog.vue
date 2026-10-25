<template>
  <div v-if="visible" class="login-overlay" @click.self="handleOverlayClick">
    <div class="login-dialog">
      <div class="login-header">
        <h2>🔐 登录 PanSou</h2>
        <p>请输入您的账号和密码</p>
      </div>
      
      <form @submit.prevent="handleLogin" class="login-form">
        <div class="form-group">
          <label>用户名</label>
          <input
            v-model="form.username"
            type="text"
            placeholder="请输入用户名"
            required
            autofocus
            autocomplete="username"
          />
        </div>
        
        <div class="form-group">
          <label>密码</label>
          <input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            required
            autocomplete="current-password"
          />
        </div>
        
        <div v-if="error" class="error-message">
          {{ error }}
        </div>
        
        <button type="submit" :disabled="loading" class="login-button">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue';
import { login } from '@/api';

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
  success: [];
}>();

const form = reactive({
  username: '',
  password: ''
});

const loading = ref(false);
const error = ref('');

watch(() => props.visible, (newVal) => {
  if (newVal) {
    error.value = '';
    form.username = '';
    form.password = '';
  }
});

const handleOverlayClick = () => {
};

const handleLogin = async () => {
  loading.value = true;
  error.value = '';
  
  try {
    const response = await login(form);
    localStorage.setItem('auth_token', response.token);
    localStorage.setItem('auth_username', response.username);
    emit('update:visible', false);
    emit('success');
  } catch (err: any) {
    error.value = err.response?.data?.error || '登录失败，请检查账号密码';
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  animation: fadeIn 0.2s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.login-dialog {
  background: hsl(var(--background));
  border-radius: 12px;
  padding: 2rem;
  width: 90%;
  max-width: 400px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s ease-out;
  border: 1px solid hsl(var(--border));
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.login-header {
  text-align: center;
  margin-bottom: 1.5rem;
}

.login-header h2 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: hsl(var(--foreground));
}

.login-header p {
  color: hsl(var(--muted-foreground));
  font-size: 0.875rem;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-group label {
  font-size: 0.875rem;
  font-weight: 500;
  color: hsl(var(--foreground));
}

.form-group input {
  padding: 0.75rem 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 0.5rem;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.875rem;
  transition: all 0.2s ease;
}

.form-group input:focus {
  outline: none;
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsla(var(--primary), 0.1);
}

.form-group input::placeholder {
  color: hsl(var(--muted-foreground));
}

.error-message {
  padding: 0.75rem;
  background: hsl(0, 84%, 95%);
  border: 1px solid hsl(0, 84%, 80%);
  color: hsl(0, 84%, 40%);
  border-radius: 0.5rem;
  font-size: 0.875rem;
  animation: shake 0.3s ease-out;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-10px); }
  75% { transform: translateX(10px); }
}

.login-button {
  padding: 0.75rem 1.5rem;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  border: none;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  margin-top: 0.5rem;
}

.login-button:hover:not(:disabled) {
  background: hsl(var(--primary) / 0.9);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px hsla(var(--primary), 0.3);
}

.login-button:active:not(:disabled) {
  transform: translateY(0);
}

.login-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 640px) {
  .login-dialog {
    padding: 1.5rem;
    max-width: 350px;
  }
  
  .login-header h2 {
    font-size: 1.25rem;
  }
}
</style>
