<template>
  <section class="studio-albums">
    <header class="studio-albums__header">
      <div>
        <p>PERSONAL GALLERY</p>
        <h1>我的相册</h1>
        <span>相册和照片归当前账号所有，可在公开主页展示。</span>
      </div>
      <button type="button" @click="newAlbum">新建相册</button>
    </header>

    <p v-if="error" class="studio-albums__notice is-error" role="alert">{{ error }}</p>
    <p v-if="notice" class="studio-albums__notice" role="status">{{ notice }}</p>

    <div class="studio-albums__layout">
      <section class="studio-albums__panel" aria-label="相册编辑">
        <h2>{{ form.id ? '编辑相册' : '创建相册' }}</h2>
        <label>名称<input v-model.trim="form.albumName" maxlength="20" placeholder="例如：旅行记录" /></label>
        <label>说明<textarea v-model.trim="form.albumDesc" maxlength="50" rows="3" placeholder="简单介绍这个相册" /></label>
        <label>封面地址<div class="studio-albums__cover-input"><input v-model.trim="form.albumCover" maxlength="255" placeholder="https://…" /><button type="button" :disabled="uploading" @click="coverInput?.click()">上传封面</button><input ref="coverInput" hidden type="file" accept="image/*" @change="uploadCover" /></div></label>
        <div v-if="form.albumCover" class="studio-albums__cover-preview"><img :src="form.albumCover" alt="相册封面预览" /></div>
        <label class="studio-albums__visibility">公开状态
          <select v-model.number="form.status"><option :value="1">公开</option><option :value="2">私有</option></select>
        </label>
        <div class="studio-albums__actions"><button type="button" :disabled="saving || !form.albumName" @click="saveAlbum">{{ saving ? '保存中…' : '保存相册' }}</button><button v-if="form.id" type="button" class="is-quiet" @click="newAlbum">取消编辑</button></div>

        <div v-if="activeAlbum" class="studio-albums__photo-tools">
          <div class="studio-albums__photo-title"><h2>{{ activeAlbum.albumName }} · 照片</h2><button type="button" class="is-quiet" @click="activeAlbum = null; photos = []">关闭</button></div>
          <p>选择图片上传，或粘贴已上传图片的网址。</p>
          <input ref="photoInput" hidden type="file" accept="image/*" multiple @change="uploadPhotos" />
          <button type="button" :disabled="uploading" @click="photoInput?.click()">{{ uploading ? '上传中…' : '上传图片' }}</button>
          <textarea v-model="photoURLs" rows="4" placeholder="每行一个 https:// 图片地址" />
          <button type="button" :disabled="savingPhotos || !photoURLs.trim()" @click="savePhotoURLs">添加图片地址</button>
          <div class="studio-albums__photos">
            <article v-for="photo in photos" :key="photo.id">
              <img :src="photo.photoSrc" :alt="photo.photoName || '相册照片'" loading="lazy" />
              <button type="button" :aria-label="`删除 ${photo.photoName || '照片'}`" @click="deletePhoto(photo.id)">删除</button>
            </article>
            <p v-if="!photos.length" class="studio-albums__empty">这个相册还没有照片。</p>
          </div>
        </div>
      </section>

      <section class="studio-albums__list" aria-label="我的相册">
        <h2>相册列表 <small>{{ albums.length }}</small></h2>
        <article v-for="album in albums" :key="album.id" class="studio-albums__card">
          <button type="button" class="studio-albums__card-main" @click="openPhotos(album)">
            <img v-if="album.albumCover" :src="album.albumCover" :alt="album.albumName" loading="lazy" />
            <span v-else class="studio-albums__cover-empty">无封面</span>
            <span><strong>{{ album.albumName }}</strong><small>{{ album.albumDesc || '暂无说明' }} · {{ album.status === 1 ? '公开' : '私有' }}</small></span>
          </button>
          <div><button type="button" @click="editAlbum(album)">编辑</button><button type="button" class="is-quiet" @click="removeAlbum(album)">删除</button></div>
        </article>
        <p v-if="!albums.length" class="studio-albums__empty">还没有相册。创建后可以上传照片。</p>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import api from '@/api/api'

type Album = { id: number; albumName: string; albumDesc: string; albumCover: string; status: number }
type Photo = { id: number; photoName: string; photoDesc: string; photoSrc: string }

const albums = ref<Album[]>([])
const photos = ref<Photo[]>([])
const activeAlbum = ref<Album | null>(null)
const photoURLs = ref('')
const coverInput = ref<HTMLInputElement | null>(null)
const photoInput = ref<HTMLInputElement | null>(null)
const loading = ref(false)
const saving = ref(false)
const savingPhotos = ref(false)
const uploading = ref(false)
const error = ref('')
const notice = ref('')
const form = reactive({ id: 0, albumName: '', albumDesc: '', albumCover: '', status: 1 })

onMounted(() => void loadAlbums())

async function loadAlbums(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const response = await api.getStudioAlbums()
    if (!response.data?.flag) throw new Error(response.data?.message || '相册加载失败')
    albums.value = Array.isArray(response.data?.data) ? response.data.data : []
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '相册加载失败'
  } finally {
    loading.value = false
  }
}

function newAlbum(): void {
  Object.assign(form, { id: 0, albumName: '', albumDesc: '', albumCover: '', status: 1 })
}

function editAlbum(album: Album): void {
  Object.assign(form, album)
  activeAlbum.value = null
  photos.value = []
}

async function saveAlbum(): Promise<void> {
  saving.value = true
  error.value = ''
  try {
    const response = await api.saveStudioAlbum({ ...form }, form.id || undefined)
    if (!response.data?.flag) throw new Error(response.data?.message || '相册保存失败')
    notice.value = '相册已保存。'
    await loadAlbums()
    newAlbum()
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '相册保存失败'
  } finally {
    saving.value = false
  }
}

async function removeAlbum(album: Album): Promise<void> {
  if (!window.confirm(`删除相册“${album.albumName}”？`)) return
  try {
    const response = await api.deleteStudioAlbum(album.id)
    if (!response.data?.flag) throw new Error(response.data?.message || '删除失败')
    if (activeAlbum.value?.id === album.id) activeAlbum.value = null
    notice.value = '相册已删除。'
    await loadAlbums()
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '删除失败'
  }
}

async function openPhotos(album: Album): Promise<void> {
  activeAlbum.value = album
  error.value = ''
  try {
    const response = await api.getStudioAlbumPhotos(album.id)
    if (!response.data?.flag) throw new Error(response.data?.message || '照片加载失败')
    photos.value = response.data?.data?.photos || []
  } catch (reason: any) {
    photos.value = []
    error.value = reason?.response?.data?.message || reason?.message || '照片加载失败'
  }
}

async function uploadCover(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploading.value = true
  try {
    const response = await api.uploadStudioAsset(file, 'album-cover')
    if (!response.data?.flag || typeof response.data?.data !== 'string') throw new Error(response.data?.message || '封面上传失败')
    form.albumCover = response.data.data
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '封面上传失败'
  } finally {
    uploading.value = false
  }
}

async function uploadPhotos(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length || !activeAlbum.value) return
  uploading.value = true
  error.value = ''
  try {
    const urls: string[] = []
    for (const file of files) {
      const response = await api.uploadStudioAsset(file, 'photo')
      if (!response.data?.flag || typeof response.data?.data !== 'string') throw new Error(response.data?.message || `${file.name} 上传失败`)
      urls.push(response.data.data)
    }
    const saved = await api.saveStudioAlbumPhotos(activeAlbum.value.id, urls)
    if (!saved.data?.flag) throw new Error(saved.data?.message || '照片保存失败')
    notice.value = `已添加 ${urls.length} 张照片。`
    await openPhotos(activeAlbum.value)
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '照片上传失败'
  } finally {
    uploading.value = false
  }
}

async function savePhotoURLs(): Promise<void> {
  if (!activeAlbum.value) return
  const urls = photoURLs.value.split(/\r?\n/).map((value) => value.trim()).filter(Boolean)
  if (!urls.length) return
  savingPhotos.value = true
  try {
    const response = await api.saveStudioAlbumPhotos(activeAlbum.value.id, urls)
    if (!response.data?.flag) throw new Error(response.data?.message || '照片添加失败')
    photoURLs.value = ''
    notice.value = `已添加 ${urls.length} 张照片。`
    await openPhotos(activeAlbum.value)
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '照片添加失败'
  } finally {
    savingPhotos.value = false
  }
}

async function deletePhoto(photoId: number): Promise<void> {
  try {
    const response = await api.deleteStudioAlbumPhotos([photoId])
    if (!response.data?.flag) throw new Error(response.data?.message || '删除失败')
    if (activeAlbum.value) await openPhotos(activeAlbum.value)
  } catch (reason: any) {
    error.value = reason?.response?.data?.message || reason?.message || '照片删除失败'
  }
}
</script>

<style scoped>
.studio-albums { display: grid; gap: 22px; }
.studio-albums__header, .studio-albums__photo-title { display: flex; justify-content: space-between; align-items: center; gap: 18px; }
.studio-albums__header p { color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.studio-albums__header h1 { margin: 4px 0 8px; font-size: 1.8rem; }
.studio-albums__header span, .studio-albums__photo-tools > p { color: var(--text-ob-dim); font-size: 12px; }
.studio-albums button { min-height: 36px; padding: 0 14px; border: 1px solid color-mix(in srgb, var(--color-ob) 40%, transparent); border-radius: 999px; background: color-mix(in srgb, var(--color-ob) 13%, transparent); color: var(--color-ob); font: inherit; font-size: 12px; cursor: pointer; }
.studio-albums button:disabled { opacity: .55; cursor: wait; }
.studio-albums button.is-quiet { border-color: var(--border-hairline); background: transparent; color: var(--text-ob-dim); }
.studio-albums__notice { padding: 12px 14px; border: 1px solid var(--border-hairline); border-radius: 12px; color: var(--text-ob-dim); }
.studio-albums__notice.is-error { border-color: #ce706b; color: #ce706b; }
.studio-albums__layout { display: grid; grid-template-columns: minmax(260px, .8fr) minmax(0, 1.2fr); gap: 18px; align-items: start; }
.studio-albums__panel, .studio-albums__list { min-width: 0; padding: 22px; border: 1px solid var(--border-hairline); border-radius: 18px; background: color-mix(in srgb, var(--background-primary-alt) 86%, transparent); }
.studio-albums h2 { margin: 0 0 16px; font-size: 1.1rem; }
.studio-albums h2 small { color: var(--text-ob-dim); font-size: 11px; }
.studio-albums label { display: grid; gap: 7px; margin-top: 14px; color: var(--text-ob-dim); font-size: 12px; }
.studio-albums input, .studio-albums textarea, .studio-albums select { width: 100%; min-height: 40px; padding: 10px 12px; border: 1px solid var(--border-hairline); border-radius: 10px; background: var(--background-primary); color: inherit; font: inherit; font-size: 12px; }
.studio-albums textarea { resize: vertical; }
.studio-albums__cover-input { display: flex; gap: 8px; }
.studio-albums__cover-input button { flex: none; }
.studio-albums__cover-preview img { width: 100%; max-height: 180px; margin-top: 12px; border-radius: 12px; object-fit: cover; }
.studio-albums__visibility select { max-width: 180px; }
.studio-albums__actions { display: flex; gap: 8px; margin-top: 18px; }
.studio-albums__card { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 12px 0; border-bottom: 1px solid var(--border-hairline); }
.studio-albums__card:last-of-type { border-bottom: 0; }
.studio-albums__card-main { display: flex; flex: 1; align-items: center; gap: 12px; min-width: 0; padding: 0 !important; border: 0 !important; background: transparent !important; color: inherit !important; text-align: left; }
.studio-albums__card-main img, .studio-albums__cover-empty { width: 72px; height: 58px; flex: none; border-radius: 10px; object-fit: cover; }
.studio-albums__cover-empty { display: grid; place-items: center; background: var(--background-primary); color: var(--text-ob-dim); font-size: 10px; }
.studio-albums__card-main span:last-child { display: grid; gap: 5px; min-width: 0; }
.studio-albums__card-main strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.studio-albums__card-main small { color: var(--text-ob-dim); font-size: 10px; }
.studio-albums__card > div { display: flex; gap: 6px; flex: none; }
.studio-albums__photo-tools { display: grid; gap: 12px; margin-top: 26px; padding-top: 20px; border-top: 1px solid var(--border-hairline); }
.studio-albums__photo-tools h2 { margin: 0; }
.studio-albums__photo-tools textarea { min-height: 90px; }
.studio-albums__photos { display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 10px; }
.studio-albums__photos article { position: relative; overflow: hidden; aspect-ratio: 1; border-radius: 12px; }
.studio-albums__photos img { width: 100%; height: 100%; object-fit: cover; }
.studio-albums__photos article button { position: absolute; right: 7px; bottom: 7px; min-height: 28px; background: rgba(14, 20, 40, .8); color: white; }
.studio-albums__empty { padding: 25px 0; color: var(--text-ob-dim); font-size: 12px; text-align: center; }
@media (max-width: 820px) { .studio-albums__layout { grid-template-columns: 1fr; } }
</style>
