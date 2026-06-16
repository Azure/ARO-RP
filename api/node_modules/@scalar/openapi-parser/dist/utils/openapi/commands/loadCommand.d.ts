import type { AnyApiDefinitionFormat, AnyObject, LoadResult, Queue, Task } from '../../../types/index.js';
import type { DereferenceOptions } from '../../../utils/dereference.js';
import type { LoadOptions } from '../../../utils/load/load.js';
import type { ValidateOptions } from '../../../utils/validate.js';
declare global {
    interface Commands {
        load: {
            task: {
                name: 'load';
                options?: LoadOptions;
            };
            result: LoadResult;
        };
    }
}
/**
 * Pass any OpenAPI document
 */
export declare function loadCommand<T extends Task[]>(previousQueue: Queue<T>, input: AnyApiDefinitionFormat, options?: LoadOptions): {
    dereference: (dereferenceOptions?: DereferenceOptions) => {
        details: () => Promise<import("../../../types/index.js").DetailsResult>;
        files: () => Promise<import("../../../types/index.js").Filesystem>;
        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
            readonly name: "load";
            readonly options: {
                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                readonly filename?: string;
                readonly filesystem?: import("../../../types/index.js").Filesystem;
                throwOnError: boolean;
            };
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
                readonly name: "load";
                readonly options: {
                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                    readonly filename?: string;
                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                    throwOnError: boolean;
                };
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
            readonly name: "load";
            readonly options: {
                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                readonly filename?: string;
                readonly filesystem?: import("../../../types/index.js").Filesystem;
                throwOnError: boolean;
            };
        }, {
            name: "filter";
            options?: import("../../filter.js").FilterCallback;
        }]>>;
        toJson: () => Promise<string>;
        toYaml: () => Promise<string>;
    };
    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
        readonly name: "load";
        readonly options: {
            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
            readonly filename?: string;
            readonly filesystem?: import("../../../types/index.js").Filesystem;
            throwOnError: boolean;
        };
    }]>>;
    upgrade: () => {
        dereference: (dereferenceOptions?: DereferenceOptions) => {
            details: () => Promise<import("../../../types/index.js").DetailsResult>;
            files: () => Promise<import("../../../types/index.js").Filesystem>;
            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                readonly name: "load";
                readonly options: {
                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                    readonly filename?: string;
                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                    throwOnError: boolean;
                };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                readonly name: "load";
                readonly options: {
                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                    readonly filename?: string;
                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                    throwOnError: boolean;
                };
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
            readonly name: "load";
            readonly options: {
                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                readonly filename?: string;
                readonly filesystem?: import("../../../types/index.js").Filesystem;
                throwOnError: boolean;
            };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                readonly name: "load";
                readonly options: {
                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                    readonly filename?: string;
                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                    throwOnError: boolean;
                };
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
                        }, {
                            name: "upgrade";
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "upgrade";
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
                        }, {
                            name: "upgrade";
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
                    }, {
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
                        }, {
                            name: "upgrade";
                        }, {
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "upgrade";
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
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
                                                                                                    readonly name: "load";
                                                                                                    readonly options: {
                                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                        readonly filename?: string;
                                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                        throwOnError: boolean;
                                                                                                    };
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
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
                                                                                                    readonly name: "load";
                                                                                                    readonly options: {
                                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                        readonly filename?: string;
                                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                        throwOnError: boolean;
                                                                                                    };
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
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
                                                                                                        readonly name: "load";
                                                                                                        readonly options: {
                                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                            readonly filename?: string;
                                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                            throwOnError: boolean;
                                                                                                        };
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
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
                                                                                                    readonly name: "load";
                                                                                                    readonly options: {
                                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                        readonly filename?: string;
                                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                        throwOnError: boolean;
                                                                                                    };
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
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
    toJson: () => Promise<string>;
    toYaml: () => Promise<string>;
    validate: (validateOptions?: ValidateOptions) => {
        dereference: (dereferenceOptions?: DereferenceOptions) => {
            details: () => Promise<import("../../../types/index.js").DetailsResult>;
            files: () => Promise<import("../../../types/index.js").Filesystem>;
            get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
                readonly name: "load";
                readonly options: {
                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                    readonly filename?: string;
                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                    throwOnError: boolean;
                };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                readonly name: "load";
                readonly options: {
                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                    readonly filename?: string;
                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                    throwOnError: boolean;
                };
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
            readonly name: "load";
            readonly options: {
                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                readonly filename?: string;
                readonly filesystem?: import("../../../types/index.js").Filesystem;
                throwOnError: boolean;
            };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                readonly name: "load";
                readonly options: {
                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                    readonly filename?: string;
                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                    throwOnError: boolean;
                };
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
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
                    readonly name: "load";
                    readonly options: {
                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                        readonly filename?: string;
                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                        throwOnError: boolean;
                    };
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
                        }, {
                            name: "validate";
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "validate";
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
                        }, {
                            name: "validate";
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
                        readonly name: "load";
                        readonly options: {
                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                            readonly filename?: string;
                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                            throwOnError: boolean;
                        };
                    }, {
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
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
                            readonly name: "load";
                            readonly options: {
                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                readonly filename?: string;
                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                throwOnError: boolean;
                            };
                        }, {
                            name: "validate";
                            options?: ValidateOptions;
                        }, {
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
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
                                readonly name: "load";
                                readonly options: {
                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                    readonly filename?: string;
                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                    throwOnError: boolean;
                                };
                            }, {
                                name: "validate";
                                options?: ValidateOptions;
                            }, {
                                name: "upgrade";
                            }, {
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
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
                                    readonly name: "load";
                                    readonly options: {
                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                        readonly filename?: string;
                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                        throwOnError: boolean;
                                    };
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
                                    name: "upgrade";
                                }, {
                                    name: "validate";
                                    options?: ValidateOptions;
                                }, {
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
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
                                        readonly name: "load";
                                        readonly options: {
                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                            readonly filename?: string;
                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                            throwOnError: boolean;
                                        };
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
                                        name: "validate";
                                        options?: ValidateOptions;
                                    }, {
                                        name: "upgrade";
                                    }, {
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
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
                                            readonly name: "load";
                                            readonly options: {
                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                readonly filename?: string;
                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                throwOnError: boolean;
                                            };
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
                                            name: "upgrade";
                                        }, {
                                            name: "validate";
                                            options?: ValidateOptions;
                                        }, {
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
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
                                                readonly name: "load";
                                                readonly options: {
                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                    readonly filename?: string;
                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                    throwOnError: boolean;
                                                };
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
                                                name: "validate";
                                                options?: ValidateOptions;
                                            }, {
                                                name: "upgrade";
                                            }, {
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
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
                                                    readonly name: "load";
                                                    readonly options: {
                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                        readonly filename?: string;
                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                        throwOnError: boolean;
                                                    };
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
                                                    name: "upgrade";
                                                }, {
                                                    name: "validate";
                                                    options?: ValidateOptions;
                                                }, {
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
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
                                                        readonly name: "load";
                                                        readonly options: {
                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                            readonly filename?: string;
                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                            throwOnError: boolean;
                                                        };
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
                                                        name: "validate";
                                                        options?: ValidateOptions;
                                                    }, {
                                                        name: "upgrade";
                                                    }, {
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
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
                                                            readonly name: "load";
                                                            readonly options: {
                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                readonly filename?: string;
                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                throwOnError: boolean;
                                                            };
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
                                                            name: "upgrade";
                                                        }, {
                                                            name: "validate";
                                                            options?: ValidateOptions;
                                                        }, {
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
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
                                                                readonly name: "load";
                                                                readonly options: {
                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                    readonly filename?: string;
                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                    throwOnError: boolean;
                                                                };
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
                                                                name: "validate";
                                                                options?: ValidateOptions;
                                                            }, {
                                                                name: "upgrade";
                                                            }, {
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
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
                                                                    readonly name: "load";
                                                                    readonly options: {
                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                        readonly filename?: string;
                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                        throwOnError: boolean;
                                                                    };
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
                                                                    name: "upgrade";
                                                                }, {
                                                                    name: "validate";
                                                                    options?: ValidateOptions;
                                                                }, {
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
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
                                                                        readonly name: "load";
                                                                        readonly options: {
                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                            readonly filename?: string;
                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                            throwOnError: boolean;
                                                                        };
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
                                                                        name: "validate";
                                                                        options?: ValidateOptions;
                                                                    }, {
                                                                        name: "upgrade";
                                                                    }, {
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
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
                                                                            readonly name: "load";
                                                                            readonly options: {
                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                readonly filename?: string;
                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                throwOnError: boolean;
                                                                            };
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
                                                                            name: "upgrade";
                                                                        }, {
                                                                            name: "validate";
                                                                            options?: ValidateOptions;
                                                                        }, {
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
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
                                                                                readonly name: "load";
                                                                                readonly options: {
                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                    readonly filename?: string;
                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                    throwOnError: boolean;
                                                                                };
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
                                                                                name: "validate";
                                                                                options?: ValidateOptions;
                                                                            }, {
                                                                                name: "upgrade";
                                                                            }, {
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
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
                                                                                    readonly name: "load";
                                                                                    readonly options: {
                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                        readonly filename?: string;
                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                        throwOnError: boolean;
                                                                                    };
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
                                                                                    name: "upgrade";
                                                                                }, {
                                                                                    name: "validate";
                                                                                    options?: ValidateOptions;
                                                                                }, {
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
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
                                                                                        readonly name: "load";
                                                                                        readonly options: {
                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                            readonly filename?: string;
                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                            throwOnError: boolean;
                                                                                        };
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
                                                                                        name: "validate";
                                                                                        options?: ValidateOptions;
                                                                                    }, {
                                                                                        name: "upgrade";
                                                                                    }, {
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
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
                                                                                                    readonly name: "load";
                                                                                                    readonly options: {
                                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                        readonly filename?: string;
                                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                        throwOnError: boolean;
                                                                                                    };
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
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
                                                                                            readonly name: "load";
                                                                                            readonly options: {
                                                                                                readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                readonly filename?: string;
                                                                                                readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                throwOnError: boolean;
                                                                                            };
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
                                                                                            name: "upgrade";
                                                                                        }, {
                                                                                            name: "validate";
                                                                                            options?: ValidateOptions;
                                                                                        }, {
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
                                                                                                    readonly name: "load";
                                                                                                    readonly options: {
                                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                        readonly filename?: string;
                                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                        throwOnError: boolean;
                                                                                                    };
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
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
                                                                                                        readonly name: "load";
                                                                                                        readonly options: {
                                                                                                            readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                            readonly filename?: string;
                                                                                                            readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                            throwOnError: boolean;
                                                                                                        };
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
                                                                                                        options?: ValidateOptions;
                                                                                                    }, {
                                                                                                        name: "upgrade";
                                                                                                    }, {
                                                                                                        name: "validate";
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
                                                                                                    readonly name: "load";
                                                                                                    readonly options: {
                                                                                                        readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                        readonly filename?: string;
                                                                                                        readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                        throwOnError: boolean;
                                                                                                    };
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
                                                                                                    options?: ValidateOptions;
                                                                                                }, {
                                                                                                    name: "upgrade";
                                                                                                }, {
                                                                                                    name: "validate";
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
                                                                                                readonly name: "load";
                                                                                                readonly options: {
                                                                                                    readonly plugins?: import("../../../utils/load/load.js").LoadPlugin[];
                                                                                                    readonly filename?: string;
                                                                                                    readonly filesystem?: import("../../../types/index.js").Filesystem;
                                                                                                    throwOnError: boolean;
                                                                                                };
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
                                                                                                name: "validate";
                                                                                                options?: ValidateOptions;
                                                                                            }, {
                                                                                                name: "upgrade";
                                                                                            }, {
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
                                                                                            validate: (validateOptions?: ValidateOptions) => /*elided*/ any;
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
};
//# sourceMappingURL=loadCommand.d.ts.map