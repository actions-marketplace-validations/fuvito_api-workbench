import '@testing-library/jest-dom/vitest'
import { configure } from '@testing-library/react'

// The whole suite runs many jsdom workers at once; a screen that loads data can take longer than the
// 1 s default to appear on a busy machine, which made findBy* queries fail now and then.
configure({ asyncUtilTimeout: 5000 })
