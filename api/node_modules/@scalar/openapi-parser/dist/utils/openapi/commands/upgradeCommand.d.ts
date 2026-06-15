import type { AnyObject, Queue, Task, UpgradeResult } from '../../../types/index.js';
import type { DereferenceOptions } from '../../../utils/dereference.js';
import type { ValidateOptions } from '../../../utils/validate.js';
declare global {
    interface Commands {
        upgrade: {
            task: {
                name: 'upgrade';
            };
            result: UpgradeResult;
        };
    }
}
/**
 * Upgrade the given OpenAPI document
 */
export declare function upgradeCommand<T extends Task[]>(previousQueue: Queue<T>): {
    dereference: (dereferenceOptions?: DereferenceOptions) => {
        details: () => Promise<import("../../../types/index.js").DetailsResult>;
        files: () => Promise<import("../../../types/index.js").Filesystem>;
        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
            name: "upgrade";
        }, {
            name: "dereference";
            options?: DereferenceOptions;
        }]>>;
        toJson: () => Promise<string>;
        toYaml: () => Promise<string>;
    };
    details: () => Promise<import("../../../types/index.js").DetailsResult>;
    files: () => Promise<import("../../../types/index.js").Filesystem>;
    filter: (callback: (specification: AnyObject) => boolean) => {
        dereference: (dereferenceOptions?: DereferenceOptions) => {
            details: () => Promise<import("../../../types/index.js").DetailsResult>;
            files: () => Promise<import("../../../types/index.js").Filesystem>;
            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                name: "upgrade";
            }, {
                name: "filter";
                options?: import("../../filter.js").FilterCallback;
            }, {
                name: "dereference";
                options?: DereferenceOptions;
            }]>>;
            toJson: () => Promise<string>;
            toYaml: () => Promise<string>;
        };
        details: () => Promise<import("../../../types/index.js").DetailsResult>;
        files: () => Promise<import("../../../types/index.js").Filesystem>;
        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
            name: "upgrade";
        }, {
            name: "filter";
            options?: import("../../filter.js").FilterCallback;
        }]>>;
        toJson: () => Promise<string>;
        toYaml: () => Promise<string>;
    };
    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
        name: "upgrade";
    }]>>;
    toJson: () => Promise<string>;
    toYaml: () => Promise<string>;
    validate: (validateOptions?: ValidateOptions) => {
        dereference: (dereferenceOptions?: DereferenceOptions) => {
            details: () => Promise<import("../../../types/index.js").DetailsResult>;
            files: () => Promise<import("../../../types/index.js").Filesystem>;
            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                name: "upgrade";
            }, {
                name: "validate";
                options?: ValidateOptions;
            }, {
                name: "dereference";
                options?: DereferenceOptions;
            }]>>;
            toJson: () => Promise<string>;
            toYaml: () => Promise<string>;
        };
        details: () => Promise<import("../../../types/index.js").DetailsResult>;
        files: () => Promise<import("../../../types/index.js").Filesystem>;
        filter: (callback: (specification: AnyObject) => boolean) => {
            dereference: (dereferenceOptions?: DereferenceOptions) => {
                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                files: () => Promise<import("../../../types/index.js").Filesystem>;
                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                    name: "upgrade";
                }, {
                    name: "validate";
                    options?: ValidateOptions;
                }, {
                    name: "filter";
                    options?: import("../../filter.js").FilterCallback;
                }, {
                    name: "dereference";
                    options?: DereferenceOptions;
                }]>>;
                toJson: () => Promise<string>;
                toYaml: () => Promise<string>;
            };
            details: () => Promise<import("../../../types/index.js").DetailsResult>;
            files: () => Promise<import("../../../types/index.js").Filesystem>;
            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                name: "upgrade";
            }, {
                name: "validate";
                options?: ValidateOptions;
            }, {
                name: "filter";
                options?: import("../../filter.js").FilterCallback;
            }]>>;
            toJson: () => Promise<string>;
            toYaml: () => Promise<string>;
        };
        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
            name: "upgrade";
        }, {
            name: "validate";
            options?: ValidateOptions;
        }]>>;
        toJson: () => Promise<string>;
        toYaml: () => Promise<string>;
        upgrade: () => {
            dereference: (dereferenceOptions?: DereferenceOptions) => {
                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                files: () => Promise<import("../../../types/index.js").Filesystem>;
                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                    name: "upgrade";
                }, {
                    name: "validate";
                    options?: ValidateOptions;
                }, {
                    name: "upgrade";
                }, {
                    name: "dereference";
                    options?: DereferenceOptions;
                }]>>;
                toJson: () => Promise<string>;
                toYaml: () => Promise<string>;
            };
            details: () => Promise<import("../../../types/index.js").DetailsResult>;
            files: () => Promise<import("../../../types/index.js").Filesystem>;
            filter: (callback: (specification: AnyObject) => boolean) => {
                dereference: (dereferenceOptions?: DereferenceOptions) => {
                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                        name: "upgrade";
                    }, {
                        name: "validate";
                        options?: ValidateOptions;
                    }, {
                        name: "upgrade";
                    }, {
                        name: "filter";
                        options?: import("../../filter.js").FilterCallback;
                    }, {
                        name: "dereference";
                        options?: DereferenceOptions;
                    }]>>;
                    toJson: () => Promise<string>;
                    toYaml: () => Promise<string>;
                };
                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                files: () => Promise<import("../../../types/index.js").Filesystem>;
                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                    name: "upgrade";
                }, {
                    name: "validate";
                    options?: ValidateOptions;
                }, {
                    name: "upgrade";
                }, {
                    name: "filter";
                    options?: import("../../filter.js").FilterCallback;
                }]>>;
                toJson: () => Promise<string>;
                toYaml: () => Promise<string>;
            };
            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                name: "upgrade";
            }, {
                name: "validate";
                options?: ValidateOptions;
            }, {
                name: "upgrade";
            }]>>;
            toJson: () => Promise<string>;
            toYaml: () => Promise<string>;
            validate: (validateOptions?: ValidateOptions) => {
                dereference: (dereferenceOptions?: DereferenceOptions) => {
                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                        name: "upgrade";
                    }, {
                        name: "validate";
                        options?: ValidateOptions;
                    }, {
                        name: "upgrade";
                    }, {
                        name: "validate";
                        options?: ValidateOptions;
                    }, {
                        name: "dereference";
                        options?: DereferenceOptions;
                    }]>>;
                    toJson: () => Promise<string>;
                    toYaml: () => Promise<string>;
                };
                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                files: () => Promise<import("../../../types/index.js").Filesystem>;
                filter: (callback: (specification: AnyObject) => boolean) => {
                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "filter";
                            options?: import("../../filter.js").FilterCallback;
                        }, {
                            name: "dereference";
                            options?: DereferenceOptions;
                        }]>>;
                        toJson: () => Promise<string>;
                        toYaml: () => Promise<string>;
                    };
                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                        name: "upgrade";
                    }, {
                        name: "validate";
                        options?: ValidateOptions;
                    }, {
                        name: "upgrade";
                    }, {
                        name: "validate";
                        options?: ValidateOptions;
                    }, {
                        name: "filter";
                        options?: import("../../filter.js").FilterCallback;
                    }]>>;
                    toJson: () => Promise<string>;
                    toYaml: () => Promise<string>;
                };
                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                    name: "upgrade";
                }, {
                    name: "validate";
                    options?: ValidateOptions;
                }, {
                    name: "upgrade";
                }, {
                    name: "validate";
                    options?: ValidateOptions;
                }]>>;
                toJson: () => Promise<string>;
                toYaml: () => Promise<string>;
                upgrade: () => {
                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "upgrade";
                        }, {
                            name: "dereference";
                            options?: DereferenceOptions;
                        }]>>;
                        toJson: () => Promise<string>;
                        toYaml: () => Promise<string>;
                    };
                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                    filter: (callback: (specification: AnyObject) => boolean) => {
                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "filter";
                                options?: import("../../filter.js").FilterCallback;
                            }, {
                                name: "dereference";
                                options?: DereferenceOptions;
                            }]>>;
                            toJson: () => Promise<string>;
                            toYaml: () => Promise<string>;
                        };
                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "upgrade";
                        }, {
                            name: "filter";
                            options?: import("../../filter.js").FilterCallback;
                        }]>>;
                        toJson: () => Promise<string>;
                        toYaml: () => Promise<string>;
                    };
                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                        name: "upgrade";
                    }, {
                        name: "validate";
                        options?: ValidateOptions;
                    }, {
                        name: "upgrade";
                    }, {
                        name: "validate";
                        options?: ValidateOptions;
                    }, {
                        name: "upgrade";
                    }]>>;
                    toJson: () => Promise<string>;
                    toYaml: () => Promise<string>;
                    validate: (validateOptions?: ValidateOptions) => {
                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "dereference";
                                options?: DereferenceOptions;
                            }]>>;
                            toJson: () => Promise<string>;
                            toYaml: () => Promise<string>;
                        };
                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                        filter: (callback: (specification: AnyObject) => boolean) => {
                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "filter";
                                    options?: import("../../filter.js").FilterCallback;
                                }, {
                                    name: "dereference";
                                    options?: DereferenceOptions;
                                }]>>;
                                toJson: () => Promise<string>;
                                toYaml: () => Promise<string>;
                            };
                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "filter";
                                options?: import("../../filter.js").FilterCallback;
                            }]>>;
                            toJson: () => Promise<string>;
                            toYaml: () => Promise<string>;
                        };
                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
                            name: "upgrade";
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }]>>;
                        toJson: () => Promise<string>;
                        toYaml: () => Promise<string>;
                        upgrade: () => {
                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "dereference";
                                    options?: DereferenceOptions;
                                }]>>;
                                toJson: () => Promise<string>;
                                toYaml: () => Promise<string>;
                            };
                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                            filter: (callback: (specification: AnyObject) => boolean) => {
                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "filter";
                                        options?: import("../../filter.js").FilterCallback;
                                    }, {
                                        name: "dereference";
                                        options?: DereferenceOptions;
                                    }]>>;
                                    toJson: () => Promise<string>;
                                    toYaml: () => Promise<string>;
                                };
                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "filter";
                                    options?: import("../../filter.js").FilterCallback;
                                }]>>;
                                toJson: () => Promise<string>;
                                toYaml: () => Promise<string>;
                            };
                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }]>>;
                            toJson: () => Promise<string>;
                            toYaml: () => Promise<string>;
                            validate: (validateOptions?: ValidateOptions) => {
                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "dereference";
                                        options?: DereferenceOptions;
                                    }]>>;
                                    toJson: () => Promise<string>;
                                    toYaml: () => Promise<string>;
                                };
                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                filter: (callback: (specification: AnyObject) => boolean) => {
                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "filter";
                                            options?: import("../../filter.js").FilterCallback;
                                        }, {
                                            name: "dereference";
                                            options?: DereferenceOptions;
                                        }]>>;
                                        toJson: () => Promise<string>;
                                        toYaml: () => Promise<string>;
                                    };
                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "filter";
                                        options?: import("../../filter.js").FilterCallback;
                                    }]>>;
                                    toJson: () => Promise<string>;
                                    toYaml: () => Promise<string>;
                                };
                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }]>>;
                                toJson: () => Promise<string>;
                                toYaml: () => Promise<string>;
                                upgrade: () => {
                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "dereference";
                                            options?: DereferenceOptions;
                                        }]>>;
                                        toJson: () => Promise<string>;
                                        toYaml: () => Promise<string>;
                                    };
                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                    filter: (callback: (specification: AnyObject) => boolean) => {
                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "filter";
                                                options?: import("../../filter.js").FilterCallback;
                                            }, {
                                                name: "dereference";
                                                options?: DereferenceOptions;
                                            }]>>;
                                            toJson: () => Promise<string>;
                                            toYaml: () => Promise<string>;
                                        };
                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "filter";
                                            options?: import("../../filter.js").FilterCallback;
                                        }]>>;
                                        toJson: () => Promise<string>;
                                        toYaml: () => Promise<string>;
                                    };
                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }]>>;
                                    toJson: () => Promise<string>;
                                    toYaml: () => Promise<string>;
                                    validate: (validateOptions?: ValidateOptions) => {
                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "dereference";
                                                options?: DereferenceOptions;
                                            }]>>;
                                            toJson: () => Promise<string>;
                                            toYaml: () => Promise<string>;
                                        };
                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                        filter: (callback: (specification: AnyObject) => boolean) => {
                                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "filter";
                                                    options?: import("../../filter.js").FilterCallback;
                                                }, {
                                                    name: "dereference";
                                                    options?: DereferenceOptions;
                                                }]>>;
                                                toJson: () => Promise<string>;
                                                toYaml: () => Promise<string>;
                                            };
                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "filter";
                                                options?: import("../../filter.js").FilterCallback;
                                            }]>>;
                                            toJson: () => Promise<string>;
                                            toYaml: () => Promise<string>;
                                        };
                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }]>>;
                                        toJson: () => Promise<string>;
                                        toYaml: () => Promise<string>;
                                        upgrade: () => {
                                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "dereference";
                                                    options?: DereferenceOptions;
                                                }]>>;
                                                toJson: () => Promise<string>;
                                                toYaml: () => Promise<string>;
                                            };
                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                            filter: (callback: (specification: AnyObject) => boolean) => {
                                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "filter";
                                                        options?: import("../../filter.js").FilterCallback;
                                                    }, {
                                                        name: "dereference";
                                                        options?: DereferenceOptions;
                                                    }]>>;
                                                    toJson: () => Promise<string>;
                                                    toYaml: () => Promise<string>;
                                                };
                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "filter";
                                                    options?: import("../../filter.js").FilterCallback;
                                                }]>>;
                                                toJson: () => Promise<string>;
                                                toYaml: () => Promise<string>;
                                            };
                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }]>>;
                                            toJson: () => Promise<string>;
                                            toYaml: () => Promise<string>;
                                            validate: (validateOptions?: ValidateOptions) => {
                                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "dereference";
                                                        options?: DereferenceOptions;
                                                    }]>>;
                                                    toJson: () => Promise<string>;
                                                    toYaml: () => Promise<string>;
                                                };
                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                filter: (callback: (specification: AnyObject) => boolean) => {
                                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "filter";
                                                            options?: import("../../filter.js").FilterCallback;
                                                        }, {
                                                            name: "dereference";
                                                            options?: DereferenceOptions;
                                                        }]>>;
                                                        toJson: () => Promise<string>;
                                                        toYaml: () => Promise<string>;
                                                    };
                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "filter";
                                                        options?: import("../../filter.js").FilterCallback;
                                                    }]>>;
                                                    toJson: () => Promise<string>;
                                                    toYaml: () => Promise<string>;
                                                };
                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }]>>;
                                                toJson: () => Promise<string>;
                                                toYaml: () => Promise<string>;
                                                upgrade: () => {
                                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "dereference";
                                                            options?: DereferenceOptions;
                                                        }]>>;
                                                        toJson: () => Promise<string>;
                                                        toYaml: () => Promise<string>;
                                                    };
                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                    filter: (callback: (specification: AnyObject) => boolean) => {
                                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "filter";
                                                                options?: import("../../filter.js").FilterCallback;
                                                            }, {
                                                                name: "dereference";
                                                                options?: DereferenceOptions;
                                                            }]>>;
                                                            toJson: () => Promise<string>;
                                                            toYaml: () => Promise<string>;
                                                        };
                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "filter";
                                                            options?: import("../../filter.js").FilterCallback;
                                                        }]>>;
                                                        toJson: () => Promise<string>;
                                                        toYaml: () => Promise<string>;
                                                    };
                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }]>>;
                                                    toJson: () => Promise<string>;
                                                    toYaml: () => Promise<string>;
                                                    validate: (validateOptions?: ValidateOptions) => {
                                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "dereference";
                                                                options?: DereferenceOptions;
                                                            }]>>;
                                                            toJson: () => Promise<string>;
                                                            toYaml: () => Promise<string>;
                                                        };
                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                        filter: (callback: (specification: AnyObject) => boolean) => {
                                                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "filter";
                                                                    options?: import("../../filter.js").FilterCallback;
                                                                }, {
                                                                    name: "dereference";
                                                                    options?: DereferenceOptions;
                                                                }]>>;
                                                                toJson: () => Promise<string>;
                                                                toYaml: () => Promise<string>;
                                                            };
                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "filter";
                                                                options?: import("../../filter.js").FilterCallback;
                                                            }]>>;
                                                            toJson: () => Promise<string>;
                                                            toYaml: () => Promise<string>;
                                                        };
                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }]>>;
                                                        toJson: () => Promise<string>;
                                                        toYaml: () => Promise<string>;
                                                        upgrade: () => {
                                                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "dereference";
                                                                    options?: DereferenceOptions;
                                                                }]>>;
                                                                toJson: () => Promise<string>;
                                                                toYaml: () => Promise<string>;
                                                            };
                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                            filter: (callback: (specification: AnyObject) => boolean) => {
                                                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "filter";
                                                                        options?: import("../../filter.js").FilterCallback;
                                                                    }, {
                                                                        name: "dereference";
                                                                        options?: DereferenceOptions;
                                                                    }]>>;
                                                                    toJson: () => Promise<string>;
                                                                    toYaml: () => Promise<string>;
                                                                };
                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "filter";
                                                                    options?: import("../../filter.js").FilterCallback;
                                                                }]>>;
                                                                toJson: () => Promise<string>;
                                                                toYaml: () => Promise<string>;
                                                            };
                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }]>>;
                                                            toJson: () => Promise<string>;
                                                            toYaml: () => Promise<string>;
                                                            validate: (validateOptions?: ValidateOptions) => {
                                                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "dereference";
                                                                        options?: DereferenceOptions;
                                                                    }]>>;
                                                                    toJson: () => Promise<string>;
                                                                    toYaml: () => Promise<string>;
                                                                };
                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                filter: (callback: (specification: AnyObject) => boolean) => {
                                                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "filter";
                                                                            options?: import("../../filter.js").FilterCallback;
                                                                        }, {
                                                                            name: "dereference";
                                                                            options?: DereferenceOptions;
                                                                        }]>>;
                                                                        toJson: () => Promise<string>;
                                                                        toYaml: () => Promise<string>;
                                                                    };
                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "filter";
                                                                        options?: import("../../filter.js").FilterCallback;
                                                                    }]>>;
                                                                    toJson: () => Promise<string>;
                                                                    toYaml: () => Promise<string>;
                                                                };
                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }]>>;
                                                                toJson: () => Promise<string>;
                                                                toYaml: () => Promise<string>;
                                                                upgrade: () => {
                                                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "dereference";
                                                                            options?: DereferenceOptions;
                                                                        }]>>;
                                                                        toJson: () => Promise<string>;
                                                                        toYaml: () => Promise<string>;
                                                                    };
                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                    filter: (callback: (specification: AnyObject) => boolean) => {
                                                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "filter";
                                                                                options?: import("../../filter.js").FilterCallback;
                                                                            }, {
                                                                                name: "dereference";
                                                                                options?: DereferenceOptions;
                                                                            }]>>;
                                                                            toJson: () => Promise<string>;
                                                                            toYaml: () => Promise<string>;
                                                                        };
                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "filter";
                                                                            options?: import("../../filter.js").FilterCallback;
                                                                        }]>>;
                                                                        toJson: () => Promise<string>;
                                                                        toYaml: () => Promise<string>;
                                                                    };
                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }]>>;
                                                                    toJson: () => Promise<string>;
                                                                    toYaml: () => Promise<string>;
                                                                    validate: (validateOptions?: ValidateOptions) => {
                                                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "dereference";
                                                                                options?: DereferenceOptions;
                                                                            }]>>;
                                                                            toJson: () => Promise<string>;
                                                                            toYaml: () => Promise<string>;
                                                                        };
                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                        filter: (callback: (specification: AnyObject) => boolean) => {
                                                                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "filter";
                                                                                    options?: import("../../filter.js").FilterCallback;
                                                                                }, {
                                                                                    name: "dereference";
                                                                                    options?: DereferenceOptions;
                                                                                }]>>;
                                                                                toJson: () => Promise<string>;
                                                                                toYaml: () => Promise<string>;
                                                                            };
                                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "filter";
                                                                                options?: import("../../filter.js").FilterCallback;
                                                                            }]>>;
                                                                            toJson: () => Promise<string>;
                                                                            toYaml: () => Promise<string>;
                                                                        };
                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }]>>;
                                                                        toJson: () => Promise<string>;
                                                                        toYaml: () => Promise<string>;
                                                                        upgrade: () => {
                                                                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "dereference";
                                                                                    options?: DereferenceOptions;
                                                                                }]>>;
                                                                                toJson: () => Promise<string>;
                                                                                toYaml: () => Promise<string>;
                                                                            };
                                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                            filter: (callback: (specification: AnyObject) => boolean) => {
                                                                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "filter";
                                                                                        options?: import("../../filter.js").FilterCallback;
                                                                                    }, {
                                                                                        name: "dereference";
                                                                                        options?: DereferenceOptions;
                                                                                    }]>>;
                                                                                    toJson: () => Promise<string>;
                                                                                    toYaml: () => Promise<string>;
                                                                                };
                                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "filter";
                                                                                    options?: import("../../filter.js").FilterCallback;
                                                                                }]>>;
                                                                                toJson: () => Promise<string>;
                                                                                toYaml: () => Promise<string>;
                                                                            };
                                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }]>>;
                                                                            toJson: () => Promise<string>;
                                                                            toYaml: () => Promise<string>;
                                                                            validate: (validateOptions?: ValidateOptions) => {
                                                                                dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "dereference";
                                                                                        options?: DereferenceOptions;
                                                                                    }]>>;
                                                                                    toJson: () => Promise<string>;
                                                                                    toYaml: () => Promise<string>;
                                                                                };
                                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                filter: (callback: (specification: AnyObject) => boolean) => {
                                                                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "filter";
                                                                                            options?: import("../../filter.js").FilterCallback;
                                                                                        }, {
                                                                                            name: "dereference";
                                                                                            options?: DereferenceOptions;
                                                                                        }]>>;
                                                                                        toJson: () => Promise<string>;
                                                                                        toYaml: () => Promise<string>;
                                                                                    };
                                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "filter";
                                                                                        options?: import("../../filter.js").FilterCallback;
                                                                                    }]>>;
                                                                                    toJson: () => Promise<string>;
                                                                                    toYaml: () => Promise<string>;
                                                                                };
                                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }]>>;
                                                                                toJson: () => Promise<string>;
                                                                                toYaml: () => Promise<string>;
                                                                                upgrade: () => {
                                                                                    dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "dereference";
                                                                                            options?: DereferenceOptions;
                                                                                        }]>>;
                                                                                        toJson: () => Promise<string>;
                                                                                        toYaml: () => Promise<string>;
                                                                                    };
                                                                                    details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                    files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                    filter: (callback: (specification: AnyObject) => boolean) => {
                                                                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "filter";
                                                                                                options?: import("../../filter.js").FilterCallback;
                                                                                            }, {
                                                                                                name: "dereference";
                                                                                                options?: DereferenceOptions;
                                                                                            }]>>;
                                                                                            toJson: () => Promise<string>;
                                                                                            toYaml: () => Promise<string>;
                                                                                        };
                                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "filter";
                                                                                            options?: import("../../filter.js").FilterCallback;
                                                                                        }]>>;
                                                                                        toJson: () => Promise<string>;
                                                                                        toYaml: () => Promise<string>;
                                                                                    };
                                                                                    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }]>>;
                                                                                    toJson: () => Promise<string>;
                                                                                    toYaml: () => Promise<string>;
                                                                                    validate: (validateOptions?: ValidateOptions) => {
                                                                                        dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "dereference";
                                                                                                options?: DereferenceOptions;
                                                                                            }]>>;
                                                                                            toJson: () => Promise<string>;
                                                                                            toYaml: () => Promise<string>;
                                                                                        };
                                                                                        details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                        files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                        filter: (callback: (specification: AnyObject) => boolean) => {
                                                                                            dereference: (dereferenceOptions?: DereferenceOptions) => {
                                                                                                details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                                files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                                get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "filter";
                                                                                                    options?: import("../../filter.js").FilterCallback;
                                                                                                }, {
                                                                                                    name: "dereference";
                                                                                                    options?: DereferenceOptions;
                                                                                                }]>>;
                                                                                                toJson: () => Promise<string>;
                                                                                                toYaml: () => Promise<string>;
                                                                                            };
                                                                                            details: () => Promise<import("../../../types/index.js").DetailsResult>;
                                                                                            files: () => Promise<import("../../../types/index.js").Filesystem>;
                                                                                            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "filter";
                                                                                                options?: import("../../filter.js").FilterCallback;
                                                                                            }]>>;
                                                                                            toJson: () => Promise<string>;
                                                                                            toYaml: () => Promise<string>;
                                                                                        };
                                                                                        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }]>>;
                                                                                        toJson: () => Promise<string>;
                                                                                        toYaml: () => Promise<string>;
                                                                                        upgrade: () => /*elided*/ any;
                                                                                    };
                                                                                };
                                                                            };
                                                                        };
                                                                    };
                                                                };
                                                            };
                                                        };
                                                    };
                                                };
                                            };
                                        };
                                    };
                                };
                            };
                        };
                    };
                };
            };
        };
    };
};
//# sourceMappingURL=upgradeCommand.d.ts.map