<script setup>
import { computed, onBeforeUnmount, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { ChatLineRound, Hide, UserFilled, View } from "@element-plus/icons-vue";

import { loginByPassword, loginBySms, registerAccount, sendSmsCode } from "../api/auth";
import { authMessages, authPageText } from "../constants/ui-text";
import { setStoredUser } from "../utils/storage";

const router = useRouter();

const authMode = ref("login");
const loginMethod = ref("password");
const pendingAction = ref("");
const passwordPasswordInputRef = ref();
const registerPasswordInputRef = ref();

const passwordVisible = reactive({
  password: false,
  register: false,
});

const smsSending = reactive({
  login: false,
  register: false,
});

const smsCountdown = reactive({
  login: 0,
  register: 0,
});

const passwordForm = reactive({
  telephone: "",
  password: "",
});

const smsForm = reactive({
  telephone: "",
  sms_code: "",
});

const registerForm = reactive({
  nickname: "",
  telephone: "",
  password: "",
  sms_code: "",
});

const passwordFormRef = ref();
const smsFormRef = ref();
const registerFormRef = ref();

const countdownTimers = {
  login: 0,
  register: 0,
};

const heroCopy = {
  eyebrow: "欢迎回来",
  title: "登录后开始聊天",
  subtitle: "用手机号登录或注册。",
};

const submitting = computed(() => pendingAction.value !== "");
const activePanelKey = computed(() =>
  authMode.value === "register" ? "register" : `login-${loginMethod.value}`,
);
const authCardTitle = computed(() =>
  authMode.value === "register" ? "创建账号" : "欢迎回来",
);
const authCardSubtitle = computed(() =>
  authMode.value === "register"
    ? "填写信息后即可完成注册。"
    : "请选择一种方式登录。",
);
const authSwitchPrompt = computed(() =>
  authMode.value === "register" ? "已有账号？" : "还没有账号？",
);
const authSwitchAction = computed(() =>
  authMode.value === "register" ? "去登录" : "去注册",
);

const loginSmsButtonText = computed(() => {
  return smsCountdown.login > 0
    ? `${smsCountdown.login}${authMessages.cooldownSuffix}`
    : authPageText.buttons.sendCode;
});

const registerSmsButtonText = computed(() => {
  return smsCountdown.register > 0
    ? `${smsCountdown.register}${authMessages.cooldownSuffix}`
    : authPageText.buttons.sendCode;
});

const phoneRule = (_rule, value, callback) => {
  if (!/^1[3-9]\d{9}$/.test(String(value).trim())) {
    callback(new Error(authMessages.invalidPhone));
    return;
  }

  callback();
};

const passwordRules = {
  telephone: [{ required: true, validator: phoneRule, trigger: "blur" }],
  password: [{ required: true, message: authMessages.passwordRequired, trigger: "blur" }],
};

const smsRules = {
  telephone: [{ required: true, validator: phoneRule, trigger: "blur" }],
  sms_code: [{ required: true, message: authMessages.smsCodeRequired, trigger: "blur" }],
};

const registerRules = {
  nickname: [{ required: true, message: authMessages.nicknameRequired, trigger: "blur" }],
  telephone: [{ required: true, validator: phoneRule, trigger: "blur" }],
  password: [{ required: true, message: authMessages.passwordRequired, trigger: "blur" }],
  sms_code: [{ required: true, message: authMessages.smsCodeRequired, trigger: "blur" }],
};

function resolveInputElement(inputComponent) {
  return inputComponent?.$el?.querySelector("input");
}

async function validateForm(formRef) {
  if (!formRef) {
    throw new Error(authMessages.formNotReady);
  }

  try {
    await formRef.validate();
  } catch (_error) {
    throw new Error("请先完成表单信息");
  }
}

async function enterHome(user, message) {
  setStoredUser(user || {});
  ElMessage.success(message);
  await router.push("/");
}

function showRequestError(error, fallbackMessage) {
  ElMessage.error(error?.message || fallbackMessage);
}

function clearCountdown(kind) {
  if (countdownTimers[kind]) {
    window.clearInterval(countdownTimers[kind]);
    countdownTimers[kind] = 0;
  }
}

function beginCountdown(kind) {
  clearCountdown(kind);
  smsCountdown[kind] = 60;
  countdownTimers[kind] = window.setInterval(() => {
    smsCountdown[kind] -= 1;

    if (smsCountdown[kind] <= 0) {
      clearCountdown(kind);
      smsCountdown[kind] = 0;
    }
  }, 1000);
}

function setAuthMode(mode) {
  authMode.value = mode;
}

function setLoginMethod(method) {
  loginMethod.value = method;
}

function togglePasswordVisibility(kind) {
  passwordVisible[kind] = !passwordVisible[kind];

  const targetRef = kind === "register" ? registerPasswordInputRef : passwordPasswordInputRef;
  resolveInputElement(targetRef.value)?.focus();
}

async function submitPasswordLogin() {
  try {
    await validateForm(passwordFormRef.value);
    pendingAction.value = "password";
    const result = await loginByPassword({ ...passwordForm });
    await enterHome(result.data, result.message || authMessages.loginSuccess);
  } catch (error) {
    showRequestError(error, authMessages.loginFailed);
  } finally {
    pendingAction.value = "";
  }
}

async function submitSmsLogin() {
  try {
    await validateForm(smsFormRef.value);
    pendingAction.value = "sms";
    const result = await loginBySms({ ...smsForm });
    await enterHome(result.data, result.message || authMessages.loginSuccess);
  } catch (error) {
    showRequestError(error, authMessages.smsLoginFailed);
  } finally {
    pendingAction.value = "";
  }
}

async function submitRegister() {
  try {
    await validateForm(registerFormRef.value);
    pendingAction.value = "register";
    const result = await registerAccount({ ...registerForm });
    await enterHome(result.data, result.message || authMessages.registerSuccess);
  } catch (error) {
    showRequestError(error, authMessages.registerFailed);
  } finally {
    pendingAction.value = "";
  }
}

async function requestSmsCode(kind) {
  const formRef = kind === "login" ? smsFormRef.value : registerFormRef.value;
  const telephone = kind === "login" ? smsForm.telephone : registerForm.telephone;

  try {
    if (!formRef) {
      throw new Error(authMessages.formNotReady);
    }

    await formRef.validateField("telephone");
    smsSending[kind] = true;
    const result = await sendSmsCode({ telephone });
    ElMessage.success(result.message || authMessages.smsCodeSent);
    beginCountdown(kind);
  } catch (error) {
    showRequestError(error, authMessages.smsCodeFailed);
  } finally {
    smsSending[kind] = false;
  }
}

onBeforeUnmount(() => {
  clearCountdown("login");
  clearCountdown("register");
});
</script>

<template>
  <main class="auth-page">
    <section class="page-shell auth-layout">
      <article class="scene-card">
        <div class="scene-copy">
          <p class="eyebrow">{{ heroCopy.eyebrow }}</p>
          <h1 class="display-title scene-copy__title">{{ heroCopy.title }}</h1>
          <p class="muted-copy scene-copy__desc">{{ heroCopy.subtitle }}</p>
        </div>
      </article>

      <article class="glass-card auth-card">
        <header class="auth-card__header">
          <div class="auth-switch" :data-mode="authMode">
            <span class="auth-switch__thumb"></span>

            <button
              type="button"
              class="auth-switch__button"
              :class="{ 'is-active': authMode === 'login' }"
              @click="setAuthMode('login')"
            >
              <el-icon><UserFilled /></el-icon>
              <span>登录</span>
            </button>

            <button
              type="button"
              class="auth-switch__button"
              :class="{ 'is-active': authMode === 'register' }"
              @click="setAuthMode('register')"
            >
              <el-icon><ChatLineRound /></el-icon>
              <span>注册</span>
            </button>
          </div>

          <div>
            <h2 class="section-title">{{ authCardTitle }}</h2>
            <p class="panel-copy auth-card__copy">{{ authCardSubtitle }}</p>
          </div>
        </header>

        <div v-if="authMode === 'login'" class="method-switch" :data-method="loginMethod">
          <span class="method-switch__thumb"></span>

          <button
            type="button"
            class="method-switch__button"
            :class="{ 'is-active': loginMethod === 'password' }"
            @click="setLoginMethod('password')"
          >
            密码登录
          </button>

          <button
            type="button"
            class="method-switch__button"
            :class="{ 'is-active': loginMethod === 'sms' }"
            @click="setLoginMethod('sms')"
          >
            短信登录
          </button>
        </div>

        <div class="auth-panel">
          <transition name="panel-swap" mode="out-in">
            <section :key="activePanelKey" class="form-panel">
              <el-form
                v-if="authMode === 'login' && loginMethod === 'password'"
                ref="passwordFormRef"
                :model="passwordForm"
                :rules="passwordRules"
                label-position="top"
                class="auth-form"
              >
                <el-form-item :label="authPageText.form.phoneLabel" prop="telephone">
                  <el-input
                    v-model="passwordForm.telephone"
                    :placeholder="authPageText.form.phonePlaceholder"
                    autocomplete="tel"
                    clearable
                    inputmode="numeric"
                    maxlength="11"
                  />
                </el-form-item>

                <el-form-item :label="authPageText.form.passwordLabel" prop="password">
                  <el-input
                    ref="passwordPasswordInputRef"
                    v-model="passwordForm.password"
                    :type="passwordVisible.password ? 'text' : 'password'"
                    :placeholder="authPageText.form.passwordPlaceholder"
                    autocomplete="current-password"
                    clearable
                    @keyup.enter="submitPasswordLogin"
                  >
                    <template #suffix>
                      <button
                        type="button"
                        class="password-vision-toggle"
                        @mousedown.prevent
                        @click="togglePasswordVisibility('password')"
                      >
                        <el-icon>
                          <View v-if="!passwordVisible.password" />
                          <Hide v-else />
                        </el-icon>
                        <span>{{ passwordVisible.password ? "隐藏" : "显示" }}</span>
                      </button>
                    </template>
                  </el-input>
                </el-form-item>

                <el-button
                  type="primary"
                  size="large"
                  class="auth-submit"
                  :loading="pendingAction === 'password'"
                  :disabled="submitting && pendingAction !== 'password'"
                  @click="submitPasswordLogin"
                >
                  {{ authPageText.buttons.login }}
                </el-button>
              </el-form>

              <el-form
                v-else-if="authMode === 'login' && loginMethod === 'sms'"
                ref="smsFormRef"
                :model="smsForm"
                :rules="smsRules"
                label-position="top"
                class="auth-form"
              >
                <el-form-item :label="authPageText.form.phoneLabel" prop="telephone">
                  <el-input
                    v-model="smsForm.telephone"
                    :placeholder="authPageText.form.phonePlaceholder"
                    autocomplete="tel"
                    clearable
                    inputmode="numeric"
                    maxlength="11"
                  />
                </el-form-item>

                <div class="inline-row">
                  <el-form-item
                    :label="authPageText.form.smsCodeLabel"
                    prop="sms_code"
                    class="inline-row__field"
                  >
                    <el-input
                      v-model="smsForm.sms_code"
                      :placeholder="authPageText.form.smsCodePlaceholder"
                      autocomplete="one-time-code"
                      clearable
                      @keyup.enter="submitSmsLogin"
                    />
                  </el-form-item>

                  <el-button
                    class="inline-row__action"
                    size="large"
                    :disabled="smsCountdown.login > 0"
                    :loading="smsSending.login"
                    @click="requestSmsCode('login')"
                  >
                    {{ loginSmsButtonText }}
                  </el-button>
                </div>

                <el-button
                  type="primary"
                  size="large"
                  class="auth-submit"
                  :loading="pendingAction === 'sms'"
                  :disabled="submitting && pendingAction !== 'sms'"
                  @click="submitSmsLogin"
                >
                  {{ authPageText.buttons.loginBySms }}
                </el-button>
              </el-form>

              <el-form
                v-else
                ref="registerFormRef"
                :model="registerForm"
                :rules="registerRules"
                label-position="top"
                class="auth-form"
              >
                <el-form-item :label="authPageText.form.nicknameLabel" prop="nickname">
                  <el-input
                    v-model="registerForm.nickname"
                    :placeholder="authPageText.form.nicknamePlaceholder"
                    autocomplete="nickname"
                    clearable
                  />
                </el-form-item>

                <el-form-item :label="authPageText.form.phoneLabel" prop="telephone">
                  <el-input
                    v-model="registerForm.telephone"
                    :placeholder="authPageText.form.phonePlaceholder"
                    autocomplete="tel"
                    clearable
                    inputmode="numeric"
                    maxlength="11"
                  />
                </el-form-item>

                <el-form-item :label="authPageText.form.passwordLabel" prop="password">
                  <el-input
                    ref="registerPasswordInputRef"
                    v-model="registerForm.password"
                    :type="passwordVisible.register ? 'text' : 'password'"
                    :placeholder="authPageText.form.passwordPlaceholder"
                    autocomplete="new-password"
                    clearable
                    @keyup.enter="submitRegister"
                  >
                    <template #suffix>
                      <button
                        type="button"
                        class="password-vision-toggle"
                        @mousedown.prevent
                        @click="togglePasswordVisibility('register')"
                      >
                        <el-icon>
                          <View v-if="!passwordVisible.register" />
                          <Hide v-else />
                        </el-icon>
                        <span>{{ passwordVisible.register ? "隐藏" : "显示" }}</span>
                      </button>
                    </template>
                  </el-input>
                </el-form-item>

                <div class="inline-row">
                  <el-form-item
                    :label="authPageText.form.smsCodeLabel"
                    prop="sms_code"
                    class="inline-row__field"
                  >
                    <el-input
                      v-model="registerForm.sms_code"
                      :placeholder="authPageText.form.smsCodePlaceholder"
                      autocomplete="one-time-code"
                      clearable
                      @keyup.enter="submitRegister"
                    />
                  </el-form-item>

                  <el-button
                    class="inline-row__action"
                    size="large"
                    :disabled="smsCountdown.register > 0"
                    :loading="smsSending.register"
                    @click="requestSmsCode('register')"
                  >
                    {{ registerSmsButtonText }}
                  </el-button>
                </div>

                <el-button
                  type="primary"
                  size="large"
                  class="auth-submit"
                  :loading="pendingAction === 'register'"
                  :disabled="submitting && pendingAction !== 'register'"
                  @click="submitRegister"
                >
                  {{ authPageText.buttons.register }}
                </el-button>
              </el-form>
            </section>
          </transition>
        </div>

        <footer class="auth-card__footer">
          <div class="switch-link-row">
            <span>{{ authSwitchPrompt }}</span>
            <button
              type="button"
              class="switch-link-row__button"
              @click="setAuthMode(authMode === 'register' ? 'login' : 'register')"
            >
              {{ authSwitchAction }}
            </button>
          </div>
        </footer>
      </article>
    </section>
  </main>
</template>

<style scoped>
.auth-page {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  background:
    radial-gradient(circle at 12% 18%, rgba(255, 255, 255, 0.94) 0, rgba(255, 255, 255, 0) 28%),
    radial-gradient(circle at 88% 10%, rgba(255, 210, 186, 0.45) 0, rgba(255, 210, 186, 0) 26%),
    linear-gradient(135deg, #fff7f1 0%, #ffe4d4 45%, #f6efe7 100%);
}

.auth-page::before,
.auth-page::after {
  position: absolute;
  content: "";
  border-radius: 999px;
  filter: blur(18px);
  pointer-events: none;
}

.auth-page::before {
  top: 160px;
  left: -80px;
  width: 240px;
  height: 240px;
  background: rgba(255, 197, 167, 0.32);
}

.auth-page::after {
  right: -90px;
  bottom: 96px;
  width: 280px;
  height: 280px;
  background: rgba(255, 232, 210, 0.54);
}

.auth-layout {
  position: relative;
  z-index: 1;
  display: grid;
  justify-items: center;
  align-content: center;
  gap: 20px;
  min-height: calc(100vh - 56px);
}

.scene-card {
  width: min(100%, 440px);
  padding: 0;
}

.scene-copy {
  display: grid;
  gap: 0;
  text-align: left;
}

.scene-copy__title {
  max-width: none;
  font-size: 36px;
  letter-spacing: 0;
  white-space: nowrap;
}

.scene-copy__desc {
  max-width: 360px;
  margin: 12px 0 0;
  font-size: 15px;
}

.auth-card {
  width: min(100%, 440px);
  position: relative;
  z-index: 1;
  padding: 28px;
  border-color: rgba(217, 168, 141, 0.26);
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 30px 56px rgba(132, 85, 55, 0.14);
  backdrop-filter: blur(18px);
}

.auth-card__header {
  display: grid;
  gap: 16px;
}

.auth-card__copy {
  margin-top: 6px;
}

.auth-switch,
.method-switch {
  position: relative;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  padding: 4px;
  border: 1px solid rgba(213, 166, 140, 0.36);
  border-radius: 999px;
  background: rgba(253, 245, 238, 0.95);
}

.auth-switch__thumb,
.method-switch__thumb {
  position: absolute;
  top: 4px;
  left: 4px;
  width: calc(50% - 4px);
  height: calc(100% - 8px);
  border-radius: 999px;
  background: linear-gradient(135deg, #ffc89c 0%, #f1aa75 100%);
  box-shadow: 0 10px 22px rgba(185, 107, 63, 0.16);
  transition: transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.auth-switch[data-mode="register"] .auth-switch__thumb,
.method-switch[data-method="sms"] .method-switch__thumb {
  transform: translateX(calc(100% + 4px));
}

.auth-switch__button,
.method-switch__button {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  padding: 10px 12px;
  border: 0;
  background: transparent;
  color: #836355;
  font: inherit;
  font-weight: 800;
  cursor: pointer;
  transition: color 0.22s ease;
}

.auth-switch__button.is-active,
.method-switch__button.is-active {
  color: #5e3421;
}

.method-switch {
  margin-top: 16px;
}

.auth-panel {
  margin-top: 16px;
}

.form-panel {
  min-height: 388px;
}

.panel-swap-enter-active,
.panel-swap-leave-active {
  transition: opacity 0.28s ease, transform 0.28s ease;
}

.panel-swap-enter-from,
.panel-swap-leave-to {
  opacity: 0;
  transform: translateY(12px) scale(0.985);
}

.auth-form {
  display: grid;
}

.auth-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.auth-form :deep(.el-form-item__label) {
  padding-bottom: 8px;
  color: #4c3428;
  font-weight: 800;
}

.auth-form :deep(.el-form-item__error) {
  font-size: 12px;
  color: #b0635f;
}

.auth-form :deep(.el-input__wrapper) {
  min-height: 52px;
  border-radius: 16px;
  background: rgba(255, 249, 245, 0.98);
  box-shadow: 0 0 0 1px rgba(220, 177, 150, 0.34) inset;
  transition: box-shadow 0.22s ease, transform 0.22s ease;
}

.auth-form :deep(.el-input__wrapper.is-focus) {
  box-shadow:
    0 0 0 1.5px rgba(185, 104, 61, 0.72) inset,
    0 14px 24px rgba(182, 110, 73, 0.12);
  transform: translateY(-1px);
}

.auth-form :deep(.el-form-item.is-error .el-input__wrapper) {
  box-shadow:
    0 0 0 1px rgba(194, 110, 103, 0.48) inset,
    0 8px 16px rgba(185, 112, 99, 0.08);
}

.auth-form :deep(.el-input__inner::placeholder) {
  color: #b59a8d;
}

.auth-form :deep(.el-input__inner) {
  font-size: 15px;
}

.password-vision-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 0 3px 8px;
  border: 0;
  background: transparent;
  color: #8f7466;
  font: inherit;
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
  opacity: 0.9;
}

.password-vision-toggle:hover {
  color: #b45b38;
}

.inline-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: end;
}

.inline-row__field {
  margin-bottom: 0;
}

.inline-row__action {
  min-width: 128px;
  margin-bottom: 18px;
}

.auth-submit {
  --el-button-bg-color: #bf663f;
  --el-button-border-color: #bf663f;
  --el-button-hover-bg-color: #ad5935;
  --el-button-hover-border-color: #ad5935;
  --el-button-active-bg-color: #9f4f2e;
  --el-button-active-border-color: #9f4f2e;
  width: 100%;
  min-height: 52px;
  margin-top: 4px;
  border-radius: 16px;
  box-shadow: 0 18px 30px rgba(182, 110, 73, 0.22);
}

.auth-submit:hover {
  transform: translateY(-1px);
}

.auth-card__footer {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid rgba(216, 171, 146, 0.34);
}

.switch-link-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 12px;
  color: #856657;
  font-size: 14px;
}

.switch-link-row__button {
  padding: 0;
  border: 0;
  background: transparent;
  color: #b65d3b;
  font: inherit;
  font-weight: 800;
  cursor: pointer;
}

@media (max-width: 760px) {
  .auth-card {
    padding: 20px 18px;
  }

  .form-panel {
    min-height: auto;
  }

  .inline-row {
    grid-template-columns: 1fr;
  }

  .inline-row__action {
    width: 100%;
    margin-bottom: 0;
  }
}

@media (max-width: 520px) {
  .scene-copy__title {
    white-space: normal;
    font-size: 28px;
  }

  .scene-copy__desc {
    max-width: none;
  }

  .auth-switch__button,
  .method-switch__button {
    font-size: 14px;
  }

  .switch-link-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
