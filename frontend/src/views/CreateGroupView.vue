<script setup>
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import {
  ArrowLeft,
  Check,
  House,
  Message,
  Setting,
  SwitchButton,
  UserFilled,
} from "@element-plus/icons-vue";

import { createGroup } from "../api/user";
import { homePageText } from "../constants/ui-text";
import { clearStoredUser, getStoredUser } from "../utils/storage";

const router = useRouter();
const currentUser = ref(getStoredUser() || {});
const formRef = ref();
const submitting = ref(false);

const createForm = reactive({
  name: "",
  notice: "",
  addMode: 0,
});

const groupRules = {
  name: [
    { required: true, message: "群名称不能为空", trigger: "blur" },
    { min: 2, max: 20, message: "群名称为 2 到 20 个字", trigger: "blur" },
  ],
  notice: [{ max: 120, message: "群公告不能超过 120 个字", trigger: "blur" }],
};

async function ensureUser() {
  const user = getStoredUser();
  if (!user?.uuid) {
    clearStoredUser();
    await router.push("/auth");
    return null;
  }

  currentUser.value = user;
  return user;
}

async function submitCreateGroup() {
  const user = await ensureUser();
  if (!user) {
    return;
  }

  try {
    await formRef.value?.validate();
    submitting.value = true;

    const result = await createGroup({
      owner_id: user.uuid,
      name: createForm.name.trim(),
      notice: createForm.notice.trim(),
      add_mode: createForm.addMode,
      avatar: "",
    });

    ElMessage.success(result.message || "创建群聊成功");
    await router.push({ name: "groups" });
  } catch (error) {
    ElMessage.error(error?.message || "创建群聊失败");
  } finally {
    submitting.value = false;
  }
}

async function goHome() {
  await router.push("/");
}

async function goGroups() {
  await router.push({ name: "groups" });
}

async function logout() {
  clearStoredUser();
  ElMessage.success(homePageText.logoutSuccess);
  await router.push("/auth");
}
</script>

<template>
  <main class="create-group-page">
    <section class="page-shell create-group-page__shell">
      <header class="glass-card create-group-navbar">
        <div class="create-group-navbar__brand">
          <button type="button" class="create-group-navbar__back" @click="goGroups">
            <el-icon><ArrowLeft /></el-icon>
          </button>
          <span class="create-group-navbar__logo">K</span>
          <div>
            <p class="create-group-navbar__eyebrow">KamaChat</p>
            <h1 class="create-group-navbar__title">创建群聊</h1>
          </div>
        </div>

        <div class="create-group-navbar__actions">
          <el-button plain @click="goHome">
            <el-icon><House /></el-icon>
            <span>返回首页</span>
          </el-button>
          <el-button type="primary" plain @click="logout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出登录</span>
          </el-button>
        </div>
      </header>

      <section class="create-group-hero">
        <div>
          <p class="create-group-hero__eyebrow">新群聊</p>
          <h2 class="create-group-hero__title">把合适的人聚到一起，开启新的聊天空间。</h2>
          <p class="create-group-hero__copy">创建后可以邀请朋友一起聊天。</p>
        </div>

        <div class="create-group-preview" aria-hidden="true">
          <span>
            <el-icon><Message /></el-icon>
          </span>
          <strong>{{ createForm.name || "新的群聊" }}</strong>
          <p>{{ createForm.notice || "群公告会显示在这里。" }}</p>
        </div>
      </section>

      <section class="create-group-layout">
        <section class="glass-card create-group-form-card">
          <header class="create-group-panel-heading">
            <div>
              <p class="create-group-panel-heading__eyebrow">群信息</p>
              <h2 class="create-group-panel-heading__title">填写群名称</h2>
            </div>
          </header>

          <el-form ref="formRef" :model="createForm" :rules="groupRules" label-position="top" class="create-group-form">
            <el-form-item label="群名称" prop="name">
              <el-input v-model="createForm.name" maxlength="20" clearable placeholder="例如：产品讨论组" />
            </el-form-item>

            <el-form-item label="加群方式">
              <el-select v-model="createForm.addMode" class="create-group-form__control">
                <el-option :value="0" label="直接加入" />
                <el-option :value="1" label="需要审核" />
              </el-select>
            </el-form-item>

            <el-form-item label="群公告" prop="notice">
              <el-input
                v-model="createForm.notice"
                type="textarea"
                :rows="5"
                maxlength="120"
                show-word-limit
                placeholder="写一段群公告"
              />
            </el-form-item>

            <div class="create-group-actions">
              <el-button size="large" @click="goGroups">
                <el-icon><ArrowLeft /></el-icon>
                <span>返回群聊</span>
              </el-button>
              <el-button type="primary" size="large" :loading="submitting" @click="submitCreateGroup">
                <el-icon><Check /></el-icon>
                <span>创建群聊</span>
              </el-button>
            </div>
          </el-form>
        </section>

        <aside class="glass-card create-group-tips">
          <article>
            <span><el-icon><UserFilled /></el-icon></span>
            <div>
              <strong>群主管理</strong>
              <p>你可以修改群信息、管理成员。</p>
            </div>
          </article>
          <article>
            <span><el-icon><Setting /></el-icon></span>
            <div>
              <strong>加群方式</strong>
              <p>直接加入适合开放群，需要审核适合更私密的群。</p>
            </div>
          </article>
        </aside>
      </section>
    </section>
  </main>
</template>

<style scoped>
.create-group-page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 14% 12%, rgba(255, 255, 255, 0.86) 0, rgba(255, 255, 255, 0) 27%),
    radial-gradient(circle at 82% 14%, rgba(255, 214, 188, 0.42) 0, rgba(255, 214, 188, 0) 27%),
    linear-gradient(135deg, #fff8f2 0%, #ffe8d8 48%, #f7f0e8 100%);
}

.create-group-page__shell {
  display: grid;
  gap: 20px;
  width: min(1040px, calc(100% - 32px));
}

.create-group-navbar,
.create-group-navbar__brand,
.create-group-navbar__actions {
  display: flex;
  align-items: center;
}

.create-group-navbar {
  justify-content: space-between;
  gap: 20px;
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(18px);
}

.create-group-navbar__brand,
.create-group-navbar__actions {
  gap: 14px;
}

.create-group-navbar__back {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border: 1px solid rgba(220, 177, 150, 0.46);
  border-radius: 14px;
  background: rgba(255, 249, 245, 0.92);
  color: #9d4d2d;
  cursor: pointer;
}

.create-group-navbar__logo {
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

.create-group-navbar__eyebrow,
.create-group-navbar__title {
  margin: 0;
}

.create-group-navbar__eyebrow {
  color: var(--kc-muted);
  font-size: 12px;
}

.create-group-navbar__title {
  font-size: 18px;
  font-weight: 800;
}

.create-group-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 340px);
  gap: 24px;
  align-items: center;
  padding: 28px;
  border: 1px solid rgba(225, 176, 146, 0.36);
  border-radius: 24px;
  background: linear-gradient(145deg, rgba(255, 250, 246, 0.96) 0%, rgba(255, 225, 205, 0.9) 100%);
  box-shadow: 0 22px 42px rgba(145, 87, 58, 0.1);
}

.create-group-hero__eyebrow,
.create-group-panel-heading__eyebrow {
  margin: 0 0 8px;
  color: #b15a34;
  font-size: 12px;
  font-weight: 800;
}

.create-group-hero__title {
  max-width: 620px;
  margin: 0;
  color: #2f211a;
  font-size: 30px;
  line-height: 1.18;
  font-weight: 800;
}

.create-group-hero__copy {
  max-width: 560px;
  margin: 14px 0 0;
  color: var(--kc-muted);
  line-height: 1.7;
}

.create-group-preview {
  display: grid;
  gap: 12px;
  padding: 20px;
  border: 1px solid rgba(255, 255, 255, 0.66);
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.62);
}

.create-group-preview span {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  border-radius: 16px;
  background: #fff0e2;
  color: #9d4d2d;
}

.create-group-preview strong {
  color: #2f211a;
  font-size: 22px;
}

.create-group-preview p {
  margin: 0;
  color: var(--kc-muted);
  line-height: 1.7;
}

.create-group-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 340px);
  gap: 20px;
}

.create-group-form-card,
.create-group-tips {
  padding: 22px;
  background: rgba(255, 255, 255, 0.84);
  backdrop-filter: blur(18px);
}

.create-group-panel-heading {
  margin-bottom: 18px;
}

.create-group-panel-heading__title {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
}

.create-group-form__control {
  width: 100%;
}

.create-group-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.create-group-form :deep(.el-form-item__label) {
  padding-bottom: 8px;
  color: #4c3428;
  font-weight: 800;
}

.create-group-form :deep(.el-input__wrapper),
.create-group-form :deep(.el-textarea__inner),
.create-group-form :deep(.el-select__wrapper) {
  min-height: 50px;
  border-radius: 16px;
  background: rgba(255, 249, 245, 0.98);
  box-shadow: 0 0 0 1px rgba(220, 177, 150, 0.34) inset;
}

.create-group-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: flex-end;
}

.create-group-actions :deep(.el-button) {
  border-radius: 14px;
}

.create-group-tips {
  display: grid;
  gap: 12px;
  align-content: start;
}

.create-group-tips article {
  display: flex;
  gap: 12px;
  padding: 14px;
  border: 1px solid rgba(232, 224, 214, 0.92);
  border-radius: 18px;
  background: rgba(255, 250, 246, 0.72);
}

.create-group-tips article > span {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: 14px;
  background: #fff0e2;
  color: #9d4d2d;
}

.create-group-tips strong,
.create-group-tips p {
  margin: 0;
}

.create-group-tips strong {
  color: #2f211a;
}

.create-group-tips p {
  margin-top: 6px;
  color: var(--kc-muted);
  line-height: 1.7;
}

@media (max-width: 860px) {
  .create-group-hero,
  .create-group-layout {
    grid-template-columns: 1fr;
  }

  .create-group-navbar,
  .create-group-navbar__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .create-group-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}

@media (max-width: 560px) {
  .create-group-page__shell {
    width: min(100% - 20px, 1040px);
  }

  .create-group-navbar,
  .create-group-hero,
  .create-group-form-card,
  .create-group-tips {
    padding: 18px;
  }
}
</style>
