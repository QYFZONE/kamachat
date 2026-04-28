<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import {
  ArrowLeft,
  Calendar,
  ChatDotRound,
  Check,
  CircleCheck,
  House,
  Iphone,
  Lock,
  Message,
  Monitor,
  RefreshRight,
  Setting,
  SwitchButton,
  Upload,
  User,
} from "@element-plus/icons-vue";

import { resolveAssetUrl } from "../api/http";
import {
  getCreatedGroups,
  getFriendList,
  getGroupSessions,
  getJoinedGroups,
  getUserInfo,
  getUserSessions,
  updateUserInfo,
  uploadAvatar,
} from "../api/user";
import { homePageText } from "../constants/ui-text";
import { clearStoredUser, getStoredUser, setStoredUser } from "../utils/storage";

const router = useRouter();
const profileFormRef = ref();
const avatarInputRef = ref();
const storedUser = ref(getStoredUser() || {});
const loading = ref(false);
const saving = ref(false);
const statsLoading = ref(false);
const selectedAvatarFile = ref(null);
const avatarPreview = ref("");
const avatarCacheKey = ref("");

const profileForm = reactive({
  uuid: "",
  nickname: "",
  telephone: "",
  email: "",
  gender: 0,
  birthday: "",
  signature: "",
  avatar: "",
});

const stats = reactive({
  friends: 0,
  groups: 0,
  sessions: 0,
  registerDays: 0,
});

const emailRule = (_rule, value, callback) => {
  const email = String(value || "").trim();

  if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    callback(new Error("邮箱格式不正确"));
    return;
  }

  callback();
};

const signatureRule = (_rule, value, callback) => {
  if (String(value || "").length > 80) {
    callback(new Error("个性签名不能超过 80 个字"));
    return;
  }

  callback();
};

const profileRules = {
  nickname: [{ required: true, message: "昵称不能为空", trigger: "blur" }],
  email: [{ validator: emailRule, trigger: "blur" }],
  signature: [{ validator: signatureRule, trigger: "blur" }],
};

const displayName = computed(() => profileForm.nickname || "Kama 用户");
const userRoleText = computed(() => (storedUser.value.is_admin === 1 ? "管理员" : "普通用户"));
const userStatusText = computed(() => (Number(storedUser.value.status) === 1 ? "已禁用" : "正常"));
const birthdayText = computed(() => formatBirthday(profileForm.birthday));
const signatureText = computed(
  () => profileForm.signature || "完善你的个性签名，让好友更容易认识你。",
);
const avatarUrl = computed(() => {
  if (avatarPreview.value) {
    return avatarPreview.value;
  }

  const url = resolveAssetUrl(profileForm.avatar || storedUser.value.avatar || "");
  if (!url || !avatarCacheKey.value || url.startsWith("data:")) {
    return url;
  }

  return `${url}${url.includes("?") ? "&" : "?"}t=${avatarCacheKey.value}`;
});

const statItems = computed(() => [
  {
    key: "friends",
    label: "好友",
    value: stats.friends,
    icon: User,
    tone: "peach",
  },
  {
    key: "groups",
    label: "群聊",
    value: stats.groups,
    icon: ChatDotRound,
    tone: "rose",
  },
  {
    key: "sessions",
    label: "聊天",
    value: stats.sessions,
    icon: Message,
    tone: "cream",
  },
  {
    key: "registerDays",
    label: "已使用",
    value: stats.registerDays,
    icon: Calendar,
    tone: "mint",
  },
]);

const securityItems = computed(() => [
  {
    key: "password",
    icon: Lock,
    title: "修改密码",
    description: "定期更换更安心",
    status: "稍后开放",
  },
  {
    key: "phone",
    icon: Iphone,
    title: "绑定手机号",
    description: maskPhone(profileForm.telephone),
    status: profileForm.telephone ? "已绑定" : "未绑定",
  },
  {
    key: "email",
    icon: CircleCheck,
    title: "绑定邮箱",
    description: profileForm.email || "未填写",
    status: profileForm.email ? "已绑定" : "未绑定",
  },
  {
    key: "device",
    icon: Monitor,
    title: "正在使用",
    description: "本机",
    status: "在线",
  },
  {
    key: "status",
    icon: Setting,
    title: "账号情况",
    description: userRoleText.value,
    status: userStatusText.value,
  },
]);

const readonlyInfoItems = computed(() => [
  {
    key: "phone",
    icon: Iphone,
    label: "手机号",
    value: maskPhone(profileForm.telephone),
  },
  {
    key: "gender",
    icon: User,
    label: "性别",
    value: genderText(profileForm.gender),
  },
  {
    key: "role",
    icon: CircleCheck,
    label: "身份",
    value: userRoleText.value,
  },
  {
    key: "status",
    icon: Setting,
    label: "账号情况",
    value: userStatusText.value,
  },
]);

function getInitials(name) {
  return String(name || "K").trim().slice(0, 1).toUpperCase();
}

function normalizeBirthday(value) {
  if (!value) {
    return "";
  }

  const text = String(value).trim();
  if (/^\d{8}$/.test(text)) {
    return text;
  }

  const dateMatch = text.match(/^(\d{4})[-/.](\d{2})[-/.](\d{2})/);
  if (dateMatch) {
    return `${dateMatch[1]}${dateMatch[2]}${dateMatch[3]}`;
  }

  return text;
}

function formatBirthday(value) {
  const birthday = normalizeBirthday(value);
  if (!birthday) {
    return "未填写";
  }

  if (/^\d{8}$/.test(birthday)) {
    return `${birthday.slice(0, 4)}-${birthday.slice(4, 6)}-${birthday.slice(6, 8)}`;
  }

  return birthday;
}

function genderText(value) {
  if (Number(value) === 0) {
    return "男";
  }

  if (Number(value) === 1) {
    return "女";
  }

  return "未填写";
}

function maskPhone(phone) {
  const text = String(phone || "");
  return text ? text.replace(/^(\d{3})\d{4}(\d{4})$/, "$1****$2") : "未绑定";
}

function parseCreatedAt(value) {
  if (!value) {
    return null;
  }

  const normalized = String(value).replace(/\./g, "-").replace(" ", "T");
  const date = new Date(normalized);
  return Number.isNaN(date.getTime()) ? null : date;
}

function countList(result) {
  return Array.isArray(result?.value?.data) ? result.value.data.length : 0;
}

function updateRegisterDays(createdAt) {
  const date = parseCreatedAt(createdAt);

  if (!date) {
    stats.registerDays = 0;
    return;
  }

  const oneDay = 24 * 60 * 60 * 1000;
  stats.registerDays = Math.max(1, Math.floor((Date.now() - date.getTime()) / oneDay) + 1);
}

function assignUserInfo(userInfo) {
  const nextUser = { ...storedUser.value, ...(userInfo || {}) };
  storedUser.value = nextUser;

  profileForm.uuid = nextUser.uuid || "";
  profileForm.nickname = nextUser.nickname || "";
  profileForm.telephone = nextUser.telephone || "";
  profileForm.email = nextUser.email || "";
  profileForm.gender = Number(nextUser.gender || 0);
  profileForm.birthday = normalizeBirthday(nextUser.birthday);
  profileForm.signature = nextUser.signature || "";
  profileForm.avatar = nextUser.avatar || "";

  updateRegisterDays(nextUser.created_at || nextUser.createdAt);
}

async function loadProfile() {
  const localUser = getStoredUser();

  if (!localUser?.uuid) {
    clearStoredUser();
    await router.push("/auth");
    return;
  }

  assignUserInfo(localUser);
  loading.value = true;

  try {
    const result = await getUserInfo(localUser.uuid);
    assignUserInfo(result.data || localUser);
    setStoredUser(storedUser.value);
  } catch (error) {
    ElMessage.warning(error?.message || "暂时无法刷新个人信息");
  } finally {
    loading.value = false;
  }
}

async function loadProfileStats() {
  if (!profileForm.uuid) {
    return;
  }

  statsLoading.value = true;

  try {
    const [friends, joinedGroups, createdGroups, userSessions, groupSessions] = await Promise.allSettled([
      getFriendList(profileForm.uuid),
      getJoinedGroups(profileForm.uuid),
      getCreatedGroups(profileForm.uuid),
      getUserSessions(profileForm.uuid),
      getGroupSessions(profileForm.uuid),
    ]);

    stats.friends = countList(friends);
    stats.groups = countList(joinedGroups) + countList(createdGroups);
    stats.sessions = countList(userSessions) + countList(groupSessions);
  } finally {
    statsLoading.value = false;
  }
}

function triggerAvatarPicker() {
  avatarInputRef.value?.click();
}

function handleAvatarChange(event) {
  const file = event.target.files?.[0];
  event.target.value = "";

  if (!file) {
    return;
  }

  if (!file.type.startsWith("image/")) {
    ElMessage.error("请选择图片文件");
    return;
  }

  if (file.size > 50000) {
    ElMessage.error("头像图片不能超过 50KB");
    return;
  }

  selectedAvatarFile.value = file;
  const reader = new FileReader();
  reader.onload = () => {
    avatarPreview.value = String(reader.result || "");
  };
  reader.readAsDataURL(file);
}

async function handleSave() {
  try {
    await profileFormRef.value?.validate();
    saving.value = true;

    let nextAvatar = profileForm.avatar;

    if (selectedAvatarFile.value) {
      const uploadResult = await uploadAvatar(profileForm.uuid, selectedAvatarFile.value);
      nextAvatar = uploadResult.avatarPath;
    }

    const payload = {
      uuid: profileForm.uuid,
      email: profileForm.email,
      nickname: profileForm.nickname.trim(),
      birthday: profileForm.birthday,
      signature: profileForm.signature,
      avatar: nextAvatar,
    };

    // TODO: Add location/school/remark when the backend UpdateUserInfoRequest supports it.
    const result = await updateUserInfo(payload);
    avatarPreview.value = "";
    selectedAvatarFile.value = null;
    avatarCacheKey.value = String(Date.now());

    try {
      const refreshed = await getUserInfo(profileForm.uuid);
      assignUserInfo(refreshed.data || { ...storedUser.value, ...payload });
    } catch (_error) {
      assignUserInfo({ ...storedUser.value, ...payload });
    }

    setStoredUser(storedUser.value);
    ElMessage.success(result.message || "保存成功");
  } catch (error) {
    ElMessage.error(error?.message || "保存失败，请稍后重试");
  } finally {
    saving.value = false;
  }
}

function handleReset() {
  avatarPreview.value = "";
  selectedAvatarFile.value = null;
  assignUserInfo(storedUser.value);
  profileFormRef.value?.clearValidate();
}

function handleSecurityAction(item) {
  // TODO: Connect password, phone, email and device management APIs when backend routes are ready.
  ElMessage.info(`${item.title}暂时还不能修改`);
}

async function goHome() {
  await router.push("/");
}

async function logout() {
  clearStoredUser();
  ElMessage.success(homePageText.logoutSuccess);
  await router.push("/auth");
}

onMounted(async () => {
  await loadProfile();
  await loadProfileStats();
});
</script>

<template>
  <main class="profile-page">
    <section class="page-shell profile-page__shell">
      <header class="glass-card profile-navbar">
        <div class="profile-navbar__brand">
          <button type="button" class="profile-navbar__back" @click="goHome">
            <el-icon><ArrowLeft /></el-icon>
          </button>
          <span class="profile-navbar__logo">K</span>
          <div>
            <p class="profile-navbar__eyebrow">KamaChat</p>
            <h1 class="profile-navbar__title">个人信息</h1>
          </div>
        </div>

        <div class="profile-navbar__actions">
          <div class="profile-navbar__user">
            <el-avatar :size="42" :src="avatarUrl || ''" class="profile-navbar__avatar">
              {{ getInitials(displayName) }}
            </el-avatar>
            <div>
              <p class="profile-navbar__name">{{ displayName }}</p>
              <p class="profile-navbar__meta">{{ userRoleText }}</p>
            </div>
          </div>
          <el-button type="primary" plain @click="logout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出登录</span>
          </el-button>
        </div>
      </header>

      <section v-loading="loading" class="profile-layout">
        <aside class="profile-sidebar">
          <section class="profile-hero-card">
            <div class="profile-hero-card__header">
              <div class="avatar-wrapper">
                <el-avatar :size="108" :src="avatarUrl || ''" class="profile-avatar">
                  {{ getInitials(displayName) }}
                </el-avatar>
                <button type="button" class="avatar-wrapper__edit" @click="triggerAvatarPicker">
                  <el-icon><Upload /></el-icon>
                  <span>更换头像</span>
                </button>
                <input
                  ref="avatarInputRef"
                  class="avatar-wrapper__input"
                  type="file"
                  accept="image/*"
                  @change="handleAvatarChange"
                />
              </div>

              <span class="profile-status">
                <span class="profile-status__dot"></span>
                在线
              </span>
            </div>

            <div class="profile-hero-card__body">
              <p class="profile-hero-card__eyebrow">手机号</p>
              <h2 class="profile-hero-card__name">{{ displayName }}</h2>
              <p class="profile-hero-card__id">{{ maskPhone(profileForm.telephone) }}</p>
              <p class="profile-hero-card__signature">{{ signatureText }}</p>
            </div>

            <div class="profile-hero-card__meta">
              <div>
                <span>账号情况</span>
                <strong>{{ userStatusText }}</strong>
              </div>
              <div>
                <span>生日</span>
                <strong>{{ birthdayText }}</strong>
              </div>
            </div>
          </section>

          <section class="glass-card stats-card">
            <header class="panel-heading">
              <p class="panel-heading__eyebrow">我的</p>
              <h2 class="panel-heading__title">聊天足迹</h2>
            </header>

            <div class="stats-grid" :class="{ 'is-loading': statsLoading }">
              <article v-for="item in statItems" :key="item.key" class="stats-item" :class="`is-${item.tone}`">
                <span class="stats-item__icon">
                  <el-icon><component :is="item.icon" /></el-icon>
                </span>
                <div>
                  <p class="stats-item__label">{{ item.label }}</p>
                  <p class="stats-item__value">{{ statsLoading ? "..." : item.value }}</p>
                </div>
              </article>
            </div>
          </section>

          <section class="glass-card security-card">
            <header class="panel-heading">
              <p class="panel-heading__eyebrow">账号安全</p>
              <h2 class="panel-heading__title">安全中心</h2>
            </header>

            <div class="security-list">
              <button
                v-for="item in securityItems"
                :key="item.key"
                type="button"
                class="security-item"
                @click="handleSecurityAction(item)"
              >
                <span class="security-item__icon">
                  <el-icon><component :is="item.icon" /></el-icon>
                </span>
                <span class="security-item__body">
                  <strong>{{ item.title }}</strong>
                  <small>{{ item.description }}</small>
                </span>
                <span class="security-item__status">{{ item.status }}</span>
              </button>
            </div>
          </section>
        </aside>

        <section class="profile-main">
          <section class="glass-card profile-info-card">
            <header class="panel-heading">
              <p class="panel-heading__eyebrow">账号信息</p>
              <h2 class="panel-heading__title">登录资料</h2>
            </header>

            <div class="readonly-info-grid">
              <article v-for="item in readonlyInfoItems" :key="item.key" class="readonly-info-item">
                <span class="readonly-info-item__icon">
                  <el-icon><component :is="item.icon" /></el-icon>
                </span>
                <div>
                  <p>{{ item.label }}</p>
                  <strong>{{ item.value }}</strong>
                </div>
              </article>
            </div>
          </section>

          <section class="glass-card profile-form-card">
            <header class="profile-form-card__header">
              <div>
                <p class="panel-heading__eyebrow">可编辑</p>
                <h2 class="panel-heading__title">个人信息</h2>
              </div>
              <p class="profile-form-card__subtitle">让好友更容易认识你。</p>
            </header>

            <el-form
              ref="profileFormRef"
              :model="profileForm"
              :rules="profileRules"
              label-position="top"
              class="profile-form"
            >
              <div class="profile-form__grid">
                <el-form-item label="昵称" prop="nickname">
                  <el-input v-model="profileForm.nickname" maxlength="20" show-word-limit clearable />
                </el-form-item>

                <el-form-item label="邮箱" prop="email">
                  <el-input v-model="profileForm.email" placeholder="填写常用邮箱" clearable />
                </el-form-item>

                <el-form-item label="生日">
                  <el-date-picker
                    v-model="profileForm.birthday"
                    class="profile-form__control"
                    type="date"
                    format="YYYY-MM-DD"
                    value-format="YYYYMMDD"
                    placeholder="选择生日"
                  />
                </el-form-item>
              </div>

              <el-form-item label="个性签名" prop="signature">
                <el-input
                  v-model="profileForm.signature"
                  type="textarea"
                  :rows="4"
                  maxlength="80"
                  show-word-limit
                  placeholder="写一句想让好友看到的话"
                />
              </el-form-item>
            </el-form>
          </section>

          <section class="glass-card action-bar">
            <div class="action-bar__hint">
              <span class="action-bar__icon">
                <el-icon><Check /></el-icon>
              </span>
              <div>
                <strong>保存信息</strong>
                <span>保存后会立即更新。</span>
              </div>
            </div>

            <div class="action-bar__buttons">
              <el-button size="large" @click="goHome">
                <el-icon><House /></el-icon>
                <span>返回首页</span>
              </el-button>
              <el-button size="large" plain @click="handleReset">
                <el-icon><RefreshRight /></el-icon>
                <span>重置</span>
              </el-button>
              <el-button type="primary" size="large" :loading="saving" @click="handleSave">
                <el-icon><Check /></el-icon>
                <span>保存修改</span>
              </el-button>
            </div>
          </section>
        </section>
      </section>
    </section>
  </main>
</template>

<style scoped>
.profile-page {
  min-height: 100vh;
  background:
    linear-gradient(135deg, rgba(255, 247, 241, 0.96) 0%, rgba(255, 232, 217, 0.88) 46%, rgba(249, 241, 232, 0.96) 100%),
    linear-gradient(180deg, #fffaf6 0%, #f7efe8 100%);
}

.profile-page__shell {
  display: grid;
  gap: 20px;
  width: min(1180px, calc(100% - 32px));
}

.profile-navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(18px);
}

.profile-navbar__brand,
.profile-navbar__actions,
.profile-navbar__user {
  display: flex;
  align-items: center;
}

.profile-navbar__brand,
.profile-navbar__user {
  gap: 14px;
}

.profile-navbar__actions {
  gap: 16px;
}

.profile-navbar__back {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border: 1px solid rgba(220, 177, 150, 0.46);
  border-radius: 14px;
  background: rgba(255, 249, 245, 0.92);
  color: #9d4d2d;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.profile-navbar__back:hover {
  transform: translateY(-1px);
  border-color: rgba(188, 90, 52, 0.46);
  box-shadow: 0 12px 20px rgba(145, 87, 58, 0.08);
}

.profile-navbar__logo {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 14px;
  background: linear-gradient(135deg, #f7bf96 0%, #ec9b66 100%);
  color: #ffffff;
  font-size: 18px;
  font-weight: 800;
  box-shadow: 0 10px 18px rgba(188, 90, 52, 0.18);
}

.profile-navbar__eyebrow,
.profile-navbar__meta {
  margin: 0;
  font-size: 12px;
  color: var(--kc-muted);
}

.profile-navbar__title,
.profile-navbar__name {
  margin: 0;
  font-size: 18px;
  font-weight: 800;
}

.profile-layout {
  display: grid;
  grid-template-columns: minmax(280px, 360px) minmax(0, 1fr);
  gap: 20px;
}

.profile-sidebar,
.profile-main {
  display: grid;
  gap: 20px;
  align-content: start;
}

.profile-hero-card {
  position: relative;
  overflow: hidden;
  padding: 24px;
  border: 1px solid rgba(225, 176, 146, 0.36);
  border-radius: 24px;
  background:
    linear-gradient(145deg, rgba(255, 247, 240, 0.95) 0%, rgba(255, 216, 190, 0.9) 100%),
    #ffffff;
  box-shadow: 0 22px 42px rgba(145, 87, 58, 0.1);
}

.profile-hero-card::before {
  position: absolute;
  top: 20px;
  right: 20px;
  width: 120px;
  height: 120px;
  border: 1px solid rgba(255, 255, 255, 0.64);
  border-radius: 34px;
  content: "";
  transform: rotate(12deg);
}

.profile-hero-card__header {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.avatar-wrapper {
  position: relative;
  width: 118px;
  height: 118px;
}

.profile-avatar {
  width: 108px;
  height: 108px;
  border: 4px solid rgba(255, 255, 255, 0.86);
  background: linear-gradient(135deg, #fff3e8 0%, #f7b17e 100%);
  color: #8f4829;
  font-size: 34px;
  font-weight: 800;
  box-shadow: 0 18px 30px rgba(145, 87, 58, 0.14);
}

.avatar-wrapper__edit {
  position: absolute;
  inset: 0 10px 10px 0;
  display: grid;
  place-items: center;
  gap: 6px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: rgba(61, 36, 24, 0.58);
  color: #ffffff;
  font: inherit;
  font-size: 12px;
  font-weight: 800;
  opacity: 0;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.avatar-wrapper:hover .avatar-wrapper__edit,
.avatar-wrapper__edit:focus-visible {
  opacity: 1;
}

.avatar-wrapper__input {
  display: none;
}

.profile-status {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.76);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.66);
  color: #6f4b3a;
  font-size: 13px;
  font-weight: 800;
}

.profile-status__dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #1f8f65;
  box-shadow: 0 0 0 4px rgba(31, 143, 101, 0.12);
}

.profile-hero-card__body {
  position: relative;
  z-index: 1;
  margin-top: 18px;
}

.profile-hero-card__eyebrow,
.panel-heading__eyebrow {
  margin: 0 0 8px;
  color: #b15a34;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
}

.profile-hero-card__name {
  margin: 0;
  color: #332017;
  font-size: 28px;
  line-height: 1.2;
  font-weight: 800;
}

.profile-hero-card__id,
.profile-hero-card__signature {
  margin: 10px 0 0;
  color: #725444;
  line-height: 1.7;
  word-break: break-word;
}

.profile-hero-card__id {
  font-size: 13px;
  font-weight: 700;
}

.profile-hero-card__signature {
  min-height: 48px;
}

.profile-hero-card__meta {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 18px;
}

.profile-hero-card__meta div {
  padding: 12px;
  border: 1px solid rgba(255, 255, 255, 0.62);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.52);
}

.profile-hero-card__meta span,
.profile-hero-card__meta strong {
  display: block;
}

.profile-hero-card__meta span {
  color: #82685a;
  font-size: 12px;
  font-weight: 700;
}

.profile-hero-card__meta strong {
  margin-top: 6px;
  color: #3f2a20;
  font-size: 14px;
}

.stats-card,
.security-card,
.profile-info-card,
.profile-form-card,
.action-bar {
  padding: 22px;
  background: rgba(255, 255, 255, 0.84);
  backdrop-filter: blur(18px);
}

.panel-heading {
  margin-bottom: 16px;
}

.panel-heading__title {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.stats-item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 82px;
  padding: 14px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.86);
}

.stats-item__icon {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 14px;
  color: #8f4829;
}

.stats-item.is-peach .stats-item__icon {
  background: #ffe0cc;
}

.stats-item.is-rose .stats-item__icon {
  background: #ffe0e0;
}

.stats-item.is-cream .stats-item__icon {
  background: #fff0ce;
}

.stats-item.is-mint .stats-item__icon {
  background: #def7e9;
}

.stats-item__label,
.stats-item__value {
  margin: 0;
}

.stats-item__label {
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.stats-item__value {
  margin-top: 4px;
  color: #2f211a;
  font-size: 22px;
  font-weight: 800;
}

.security-list {
  display: grid;
  gap: 10px;
}

.security-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  width: 100%;
  min-height: 68px;
  padding: 12px;
  border: 1px solid rgba(232, 224, 214, 0.9);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.security-item:hover {
  transform: translateY(-1px);
  border-color: rgba(188, 90, 52, 0.3);
  box-shadow: 0 12px 22px rgba(145, 87, 58, 0.08);
}

.security-item__icon {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 14px;
  background: #fff0e2;
  color: #9d4d2d;
}

.security-item__body {
  min-width: 0;
}

.security-item__body strong,
.security-item__body small {
  display: block;
}

.security-item__body strong {
  color: #2f211a;
  font-size: 14px;
}

.security-item__body small {
  overflow: hidden;
  margin-top: 5px;
  color: var(--kc-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.security-item__status {
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(255, 235, 220, 0.9);
  color: #9d4d2d;
  font-size: 12px;
  font-weight: 800;
}

.readonly-info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.readonly-info-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  min-height: 72px;
  padding: 12px;
  border: 1px solid rgba(232, 224, 214, 0.9);
  border-radius: 16px;
  background: rgba(255, 250, 246, 0.72);
}

.readonly-info-item__icon {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: 14px;
  background: #fff0e2;
  color: #9d4d2d;
}

.readonly-info-item p,
.readonly-info-item strong {
  margin: 0;
}

.readonly-info-item p {
  color: var(--kc-muted);
  font-size: 12px;
  font-weight: 700;
}

.readonly-info-item strong {
  display: block;
  margin-top: 4px;
  overflow: hidden;
  color: #2f211a;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-form-card__header {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 20px;
}

.profile-form-card__subtitle {
  max-width: 280px;
  margin: 0;
  color: var(--kc-muted);
  line-height: 1.7;
  text-align: right;
}

.profile-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 18px;
}

.profile-form__control {
  width: 100%;
}

.profile-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.profile-form :deep(.el-form-item__label) {
  padding-bottom: 8px;
  color: #4c3428;
  font-weight: 800;
}

.profile-form :deep(.el-input__wrapper),
.profile-form :deep(.el-textarea__inner),
.profile-form :deep(.el-select__wrapper) {
  min-height: 50px;
  border-radius: 16px;
  background: rgba(255, 249, 245, 0.98);
  box-shadow: 0 0 0 1px rgba(220, 177, 150, 0.34) inset;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;
}

.profile-form :deep(.el-textarea__inner) {
  padding-top: 12px;
  line-height: 1.7;
}

.profile-form :deep(.el-input__wrapper.is-focus),
.profile-form :deep(.el-select__wrapper.is-focused),
.profile-form :deep(.el-textarea__inner:focus) {
  box-shadow:
    0 0 0 1.5px rgba(185, 104, 61, 0.72) inset,
    0 14px 24px rgba(182, 110, 73, 0.12);
  transform: translateY(-1px);
}

.profile-form :deep(.el-form-item.is-error .el-input__wrapper),
.profile-form :deep(.el-form-item.is-error .el-textarea__inner) {
  box-shadow:
    0 0 0 1px rgba(194, 110, 103, 0.48) inset,
    0 8px 16px rgba(185, 112, 99, 0.08);
}

.profile-form :deep(.el-input__inner::placeholder),
.profile-form :deep(.el-textarea__inner::placeholder) {
  color: #b59a8d;
}

.profile-form :deep(.el-input.is-disabled .el-input__wrapper),
.profile-form :deep(.el-select.is-disabled .el-select__wrapper) {
  background: rgba(247, 240, 234, 0.82);
  box-shadow: 0 0 0 1px rgba(222, 211, 201, 0.86) inset;
}

.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.action-bar__hint,
.action-bar__buttons {
  display: flex;
  align-items: center;
  gap: 12px;
}

.action-bar__icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 16px;
  background: #fff0e2;
  color: #9d4d2d;
}

.action-bar__hint strong,
.action-bar__hint span {
  display: block;
}

.action-bar__hint strong {
  color: #2f211a;
}

.action-bar__hint span {
  margin-top: 4px;
  color: var(--kc-muted);
  font-size: 13px;
}

.action-bar__buttons {
  justify-content: flex-end;
  flex-wrap: wrap;
}

.action-bar__buttons :deep(.el-button) {
  min-height: 44px;
  padding-inline: 18px;
  border-radius: 14px;
}

@media (max-width: 1040px) {
  .profile-layout {
    grid-template-columns: 1fr;
  }

  .profile-sidebar {
    grid-template-columns: 1fr 1fr;
  }

  .profile-hero-card {
    grid-column: 1 / -1;
  }
}

@media (max-width: 780px) {
  .profile-navbar,
  .profile-navbar__actions,
  .profile-form-card__header,
  .action-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .profile-navbar__actions {
    width: 100%;
  }

  .profile-form-card__subtitle {
    max-width: none;
    text-align: left;
  }

  .profile-form__grid,
  .readonly-info-grid,
  .profile-sidebar,
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .action-bar__buttons {
    justify-content: stretch;
  }

  .action-bar__buttons :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}

@media (max-width: 560px) {
  .profile-page__shell {
    width: min(100% - 20px, 1180px);
  }

  .profile-navbar,
  .profile-hero-card,
  .stats-card,
  .security-card,
  .profile-info-card,
  .profile-form-card,
  .action-bar {
    padding: 18px;
  }

  .profile-navbar__brand,
  .profile-navbar__user {
    align-items: flex-start;
  }

  .profile-navbar__brand {
    flex-wrap: wrap;
  }

  .profile-hero-card__header,
  .profile-hero-card__meta {
    grid-template-columns: 1fr;
  }

  .profile-hero-card__header {
    flex-direction: column;
  }

  .profile-hero-card__name {
    font-size: 24px;
  }
}
</style>
